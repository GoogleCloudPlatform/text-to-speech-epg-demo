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
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// uploadAudioFile takes a hash value of the TTS Payload, a []byte audio payload, and writes the file to the root of the GCS bucket defined in the gcsBucketName variable using the hash as a file name. Returns an error object.
func uploadAudioFile(textPayloadFileName string, data []byte) error {

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	// Upload an object with storage.Writer.
	wc := client.Bucket(gcsBucketName).Object(textPayloadFileName).NewWriter(ctx)
	wc.Write(data)
	if err := wc.Close(); err != nil {
		return err
	}

	return nil

}

// checkGCSFilePrescence takes a file name, and checks if the file already exists in the GCS bucket defined in the gcsBucketName variable
func checkGCSFilePrescence(fileString string) (bool, error) {

	// Define a new Storage Client
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return false, err
	}

	// Check if the file exists in the GCS Bucket
	_, err = client.Bucket(gcsBucketName).Object(fileString).Attrs(ctx)

	// Catch a ErrObjectNotExist error, which indicates the file does not already exist, and return error
	if err == storage.ErrObjectNotExist {
		return false, nil
	}

	// If another error, return the error for handling
	if err != nil {
		return false, err
	}

	// If no errors, the file exists so return true
	return true, nil

}

// signURL creates a signed URL for an audio file
func signURL(url string, expiration time.Time) (string, error) {

	decodedKey, err := decodeKey()
	if err != nil {
		return "", err
	}

	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url += sep
	url += fmt.Sprintf("Expires=%d", expiration.Unix())
	url += fmt.Sprintf("&KeyName=%s", cloudCDNSignedURLKeyName)

	mac := hmac.New(sha1.New, decodedKey)
	mac.Write([]byte(url))
	sig := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	url += fmt.Sprintf("&Signature=%s", sig)
	return url, nil

}

// decodeKey reads the base64url-encoded key file and decodes it.
func decodeKey() ([]byte, error) {

	b := cloudCDNURLSigningKey

	d := make([]byte, base64.URLEncoding.DecodedLen(len(b)))
	n, err := base64.URLEncoding.Decode(d, b)
	if err != nil {
		return nil, err
	}
	return d[:n], nil
}
