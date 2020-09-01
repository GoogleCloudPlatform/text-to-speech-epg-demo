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
	"context"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

// generateAudio sends textPayload to the TTS API, and calls uploadAudioFile() to store the result in GCS.
func generateAudio(textPayload string, textPayloadFileName string, sessionKey string, voiceGender string, voiceLangageCode string) error {

	// Initiate STT Client
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return err
	}

	// Set SSML Gender
	voiceGenderConfig := texttospeechpb.SsmlVoiceGender_NEUTRAL
	if voiceGender == "male" {
		voiceGenderConfig = texttospeechpb.SsmlVoiceGender_MALE
	} else if voiceGender == "female" {
		voiceGenderConfig = texttospeechpb.SsmlVoiceGender_FEMALE
	}

	// Perform the text-to-speech request on the text input
	ttsReq := texttospeechpb.SynthesizeSpeechRequest{

		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: textPayload},
		},

		// Build the voice request with the defined parameters
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: voiceLangageCode,
			SsmlGender:   voiceGenderConfig,
		},

		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	// Make the request and handle any errors
	resp, err := client.SynthesizeSpeech(ctx, &ttsReq)
	if err != nil {
		return err
	}

	// Upload the file to GCS
	err = uploadAudioFile(textPayloadFileName, resp.AudioContent)

	// Return an error if there is a problem uploading to GCS
	if err != nil {
		return err
	}

	// Else return a nil error
	return nil

}
