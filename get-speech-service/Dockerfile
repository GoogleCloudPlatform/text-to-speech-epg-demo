# Copyright 2020 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.14

# Move to working directory /build
WORKDIR /build

# Copy the code into the container
COPY src/go.mod .
COPY src/go.sum .

# Install Dependencies
RUN go mod download -x

# Copy app
COPY src/ .

# Build the application
RUN go build -o main .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/main .

# Env
ENV CLOUD_CDN_SIGNING_KEY_SECRET_NAME=get-speech-service-cdn-signing-key
ENV CLOUD_CDN_SIGNED_URL_KEY_NAME=get-speech-service-signed-url-key

# Command to run when starting the container
CMD ["/dist/main"]