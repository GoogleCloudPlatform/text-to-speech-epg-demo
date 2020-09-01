// Copyright 2020 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	loggingEncoder               *json.Encoder
	projectID                    string
	projectNumber                string
	gcsBucketName                string
	cloudCDNSigningKeySecretName string
	cloudCDNSignedURLKeyName     string
	cloudCDNURLSigningKey        []byte

	cloudCDNEndpointFQDN     string
	defaultVoiceLanguageCode string
	defaultVoiceGender       string
)

// httpErrorResponse used for JSON error messages
type httpErrorResponse struct {
	HTTPCode int    `json:"httpCode"`
	Message  string `json:"message"`
}

func main() {

	// Define bool for checking presence of environment variables
	var ok bool

	// Get Project ID from Environment
	projectID, ok = os.LookupEnv("GOOGLE_CLOUD_PROJECT")
	if !ok {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	// Get Project Number from Environment
	projectNumber, ok = os.LookupEnv("GOOGLE_CLOUD_PROJECT_NUMBER")
	if !ok {
		log.Fatal("GOOGLE_CLOUD_PROJECT_NUMBER environment variable not set")
	}

	// Get the Cloud Run $PORT variable from the environment, deafault to 80
	listenPort, ok := os.LookupEnv("PORT")
	if !ok {
		listenPort = "80"
	}

	// Get GCS Bucket name from Environment
	gcsBucketName, ok = os.LookupEnv("GCS_BUCKET_NAME")
	if !ok {
		log.Fatal("GCS_BUCKET_NAME environment variable not set")
	}

	// Get the Secret Name for the Cloud CDN URL Signing Key
	cloudCDNSigningKeySecretName, ok = os.LookupEnv("CLOUD_CDN_SIGNING_KEY_SECRET_NAME")
	if !ok {
		log.Fatal("CLOUD_CDN_SIGNING_KEY_SECRET_NAME environment variable not set")
	}

	// Get the Key Name for the signed-url-key
	cloudCDNSignedURLKeyName, ok = os.LookupEnv("CLOUD_CDN_SIGNED_URL_KEY_NAME")
	if !ok {
		log.Fatal("CLOUD_CDN_SIGNED_URL_KEY_NAME environment variable not set")
	}

	// Get the FQDN for the Cloud CDN Endpoint
	cloudCDNEndpointFQDN, ok = os.LookupEnv("CLOUD_CDN_ENDPOINT_FQDN")
	if !ok {
		log.Fatal("CLOUD_CDN_ENDPOINT_FQDN environment variable not set")
	}

	// Get the default Language Code
	defaultVoiceLanguageCode, ok = os.LookupEnv("DEFAULT_LANGUAGE_CODE")
	if !ok {
		defaultVoiceLanguageCode = "en-GB"
	}

	// Get the default Voice Gender
	defaultVoiceGender, ok = os.LookupEnv("DEFAULT_VOICE_GENDER")
	if !ok {
		defaultVoiceGender = "neutral"
	} else {
		defaultVoiceGender = strings.ToLower(defaultVoiceGender)
	}

	// Get the Cloud CDN Signing Key
	var err error
	cloudCDNURLSigningKey, err = getGoogleSecret(cloudCDNSigningKeySecretName)
	if err != nil {
		log.Fatal("Unable to fetch Cloud CDN signing Key: " + err.Error())
	}

	// Define a new JSON Encoder for logging
	loggingEncoder = json.NewEncoder(os.Stdout)

	// Define request handlers and start web server
	http.HandleFunc("/", httpDefaultHandler)
	http.HandleFunc("/getSpeech", getSpeechHandler)
	http.ListenAndServe(":"+listenPort, nil)

}

// setHeaders is used for adding common headers to a http.ResponseWriter and handling CORS Preflight requests
func setHeaders(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST,OPTIONS")

	// If this is a CORS preflight request then return a response immediately with a 204 No Content
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
	}

}

// generateErrorResponse is used for preparing an error response
func generateErrorResponse(w http.ResponseWriter, r *http.Request, httpResponseCode int, errorMessage string, errorObj error) {

	// Define a new JSON Encoder
	jsonEncoder := json.NewEncoder(w)

	// Create Response
	response := httpErrorResponse{HTTPCode: httpResponseCode, Message: errorMessage}

	// Log the Response
	jsonLogRequest(httpResponseCode, r.URL.Path, r.RemoteAddr, errorObj)

	// Return the Response
	(w).WriteHeader(httpResponseCode)
	if err := jsonEncoder.Encode(&response); err != nil {
		panic(err)
	}

}

// returnErrorResponse is used for preparing an error response
func generateSuccessResponse(w http.ResponseWriter, r *http.Request, signedURL string, servedByCache bool) {

	// httpSuccessResponse is a struct used for returning HTTP responses
	type httpSuccessResponse struct {
		HTTPCode      int    `json:"httpCode"`
		AudioURL      string `json:"audioURL"`
		ServedByCache bool   `json:"servedByCache"`
	}

	// Define a new JSON Encoder
	jsonEncoder := json.NewEncoder(w)

	// Create Response
	response := httpSuccessResponse{HTTPCode: http.StatusOK, AudioURL: signedURL, ServedByCache: servedByCache}

	// Log the Response
	jsonLogRequest(http.StatusOK, r.URL.Path, r.RemoteAddr, nil)

	// Return the Response
	w.WriteHeader(http.StatusOK)
	if err := jsonEncoder.Encode(&response); err != nil {
		panic(err)
	}

}

// getSpeechHandler is the HTTP Handler for the /getSpeech endpoint
func getSpeechHandler(w http.ResponseWriter, r *http.Request) {

	// Set standard HTTP Headers including CORS
	setHeaders(w, r)

	// If OPTIONS - return the response
	if r.Method == "OPTIONS" {
		return
	}

	// Return an error of this is not a HTTP Post request
	if r.Method != http.MethodPost {
		// Return an error response
		generateErrorResponse(w, r, http.StatusMethodNotAllowed, "This endpoint only accepts POST requests", nil)
		return
	}

	// Define a new requestDecoder
	requestDecoder := json.NewDecoder(r.Body)

	// httpGetSpeechRequest contains the payload to be synthesized
	type httpGetSpeechRequest struct {
		TextPayload       string
		VoiceLanguageCode string
		VoiceGender       string
		SessionKey        string
	}

	// Decode the Payload
	var d httpGetSpeechRequest
	err := requestDecoder.Decode(&d)
	if err != nil {
		// Return an error response
		generateErrorResponse(w, r, http.StatusInternalServerError, "Invalid request body", err)
		return
	}

	// If TextPayload is missing from the JSON Body return an error
	if d.TextPayload == "" {
		// Return an error response
		generateErrorResponse(w, r, http.StatusBadRequest, "The 'TextPayload' field was missing from the request body", err)
		return
	}

	// Prepare VoiceLangaugeCode
	submittedLanguageCode := defaultVoiceLanguageCode
	if d.VoiceLanguageCode != "" {
		submittedLanguageCode = d.VoiceLanguageCode
	}

	// Prepare VoiceGender
	permittedVoiceGenders := []string{"male", "female", "neutral"}
	var submittedVoiceGender string
	if d.VoiceGender == "" {
		submittedVoiceGender = defaultVoiceGender
	} else if isStringInSlice(d.VoiceGender, permittedVoiceGenders) == false {
		generateErrorResponse(w, r, http.StatusBadRequest, "The 'VoiceGender' specified was invalid", err)
		return
	} else {
		submittedVoiceGender = strings.ToLower(d.VoiceGender)
	}

	// Get the raw audio URL
	audioURL, audioCached, err := fetchAudioURL(d.SessionKey, d.TextPayload, submittedVoiceGender, submittedLanguageCode)
	if err != nil {
		// Return an error response
		generateErrorResponse(w, r, http.StatusInternalServerError, "There was an error fetching the audio URL", err)
		return
	}

	// Generate a Signed URL
	signedURL, err := signURL(cloudCDNEndpointFQDN+audioURL, time.Now().Add(time.Hour*24))
	if err != nil {
		// Return an error response
		generateErrorResponse(w, r, http.StatusInternalServerError, "There was an error fetching the audio URL", err)
		return
	}

	// Return the Response
	generateSuccessResponse(w, r, signedURL, audioCached)
	return

}

// fetchAudioURL is responsible for checking if there is already a generated file in GCS, if so, it returns the file name, if false, it generates it and returns the file name
func fetchAudioURL(sessionKey string, textPayload string, voiceGender string, voiceLanguageCode string) (string, bool, error) {

	// uniqueRequestIdentifier uniquely identifes a request by combining all of the variables used into a single string
	uniqueRequestIdentifier := sessionKey + textPayload + voiceGender + voiceLanguageCode

	// Generate a hash of the uniqueRequestIdentifier
	textPayloadHash := generateHash(uniqueRequestIdentifier)

	// textPayloadFileName is the full file name as expected in GCS (Hash + .mp3)
	textPayloadFileName := textPayloadHash + ".mp3"

	// Check if the hash already exists in GCS
	gcsFileExists, err := checkGCSFilePrescence(textPayloadFileName)
	if err != nil {
		return "", false, err
	}

	// Check if the file exists, if it does, return the file name. If not, generate then return the file name
	if gcsFileExists == false {

		// Generate the Audio
		err := generateAudio(textPayload, textPayloadFileName, sessionKey, voiceGender, voiceLanguageCode)
		if err != nil {
			return "", false, err
		}

		// Return the file name and a false cache match
		return textPayloadFileName, false, nil

	} else {

		// Return the file name and a true cache match
		return textPayloadFileName, true, nil

	}

}

// httpDefaultHandler returns a 404 for unmatched routes
func httpDefaultHandler(w http.ResponseWriter, r *http.Request) {

	// Set the Response Type to Application JSON
	w.Header().Set("Content-Type", "application/json")

	// Define a new JSON Encoder
	var jsonEncoder = json.NewEncoder(w)

	// Define a response
	response := httpErrorResponse{HTTPCode: http.StatusNotFound, Message: "Not Found"}

	// Log the Response
	jsonLogRequest(http.StatusNotFound, r.URL.Path, r.RemoteAddr, nil)

	// Return the Response
	w.WriteHeader(http.StatusNotFound)
	if err := jsonEncoder.Encode(&response); err != nil {
		panic(err)
	}
	return

}
