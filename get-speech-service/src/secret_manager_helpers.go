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

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// getGoogleSecret fetches a string value from Google Secrets Manager, returns a value or an error
func getGoogleSecret(secretName string) ([]byte, error) {

	// Initiate the Secrets Manager Client
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	// Define the Secret Name
	secretFullPath := "projects/" + projectNumber + "/secrets/" + secretName + "/versions/latest"

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretFullPath,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, err
	}

	return result.Payload.Data, nil

}
