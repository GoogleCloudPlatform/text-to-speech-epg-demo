# get-speech-service

The get-speech-service is a web service written in Golang that is responsible for handling requests for speech synthesis.

Clients send a POST request to the `/getSpeech` endpoint containing the text that needs to be synthesised, alongside some additional configuration parameters.

The get-speech-service then sends the text from the request to the Google Cloud Text-to-Speech Service and saves the resulting audio file in Google Cloud Storage.

Finally, a time-bound Signed URL is generated for the resulting audio file which is returned to the client to be played to the user.

On each request, the get-speech-service also checks if a transcription for the requested text payload (and associated configuration) already exists. If so, a new Signed URL is immediately generated and returned to the client for the existing file, avoiding the need to re-synthesize the audio. This has significant performance benefits and provides cost savings.

# Endpoints

## POST /getSpeech

### Request
```
{
  "TextPayload": "Hello, World!",
  "SessionKey": "MySessionKey",
  "VoiceGender": "male",
  "VoiceLanguageCode": "en-GB",
}
```

| Key                                      | Description                                                                                                                                                                                                  |
| ---------------------------              |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TextPayload                              | **(Required)** The text payload to be synthesised. Supports [SSML](https://cloud.google.com/text-to-speech/docs/ssml).                                                                                       |
| SessionKey                               | **(Optional)** (Default: "") The SessionKey is used as part of the hash to identify if audio already exists for a TextPayload. This is used for the demo environment to provide separate caches per user.    |
| VoiceGender                              | **(Optional)** (Default: "neutral") The [Voice Gender](https://cloud.google.com/text-to-speech/docs/voices) to use. Options: <male/female/neutral>.                                                                                                                 |
| VoiceLanguageCode                        | **(Optional)** (Default: "en-GB") The [Voice Language](https://cloud.google.com/text-to-speech/docs/voices) to use.                                                                                          |

### Response
```
{
    "httpCode": 200,
    "audioURL": "https://cdn.epg-text-to-speech.demos.maynard.io/576dbd34625254a1c4797b085efc6ba555b5e8843cf7c8d731ec3f40c2c53782e1460807c5b29a298cab1f94173c932da556b914d5b71879a8b1d41c1f55c4a1.mp3?Expires=1596281791&KeyName=get-speech-service-signed-url-key&Signature=3eH16nxNNrFDb0KRtzhX_tvJaHI=",
    "servedByCache": true
}
```

| Key                                      | Description                                                                                                                                                                                                  |
| ---------------------------              |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| httpCode                                 | The HTTP Code of the response.                                                                                                                                                                               |
| audioURL                                 | The Cloud CDN [Signed URL](https://cloud.google.com/cdn/docs/using-signed-urls) to access the synthesised audio.                                                                                             |
| servedByCache                            | Indicates if the request was served from the cache in GCS (if true, this request has already been processed previously).                                                                                     |

# Environment Variables

| Environment Variable                     | Description                                                                                                                                                          |
| ---------------------------              |--------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PORT                                     | **(Optional)** The port the **get-speech-service** should listen on (Cloud Run automatically sets this Environment Variable).                                        |
| GOOGLE_APPLICATION_CREDENTIALS           | **(Local Development Only)** Path within the container to the Service Account Key (see [Developing Locally](#developing-locally)).                                 |
| GOOGLE_CLOUD_PROJECT                     | **(Required)** Google Cloud Project Name.                                                                                                                            |
| GOOGLE_CLOUD_PROJECT_NUMBER              | **(Required)** Google Cloud Project Number.                                                                                                                          |
| GCS_BUCKET_NAME                          | **(Required** GCS Bucket name where the TTS audio files will be stored.                                                                                              |
| CLOUD_CDN_SIGNING_KEY_SECRET_NAME        | **(Required - Default: get-speech-service-cdn-signing-key)** The Secrets Manager Secret Name that contains the Cloud CDN Signing Key.                                |
| CLOUD_CDN_SIGNED_URL_KEY_NAME            | **(Required - Default: get-speech-service-signed-url-key)** Signed URL Key Name as configured through `gcloud compute backend-buckets add-signed-url-key`.                                                                                                                                                                                              |
| CLOUD_CDN_ENDPOINT_FQDN                  | **(Required)** FQDN for the Cloud CDN Endpoint including trailing `/` (example: https://cdn.demos.maynard.io/).                                                      |
| DEFAULT_LANGUAGE_CODE                    | **(Optional - Default: en-GB)** Default [language code](https://cloud.google.com/text-to-speech/docs/voices) to use if not specified in the user request.            |
| DEFAULT_VOICE_GENDER                     | **(Optional - Default: neutral)** Default [voice gender](https://cloud.google.com/text-to-speech/docs/voices) to use in the request. Options: <male/female/neutral>. |

# Deployment

Continuous deployment can be set up through [Cloud Build](https://cloud.google.com/cloud-build). A sample [cloudbuild.yaml](cloudbuild.yaml) file can be found in this directory. See [here](https://cloud.google.com/cloud-build/docs/automating-builds/create-manage-triggers) for instructions on how to setup a Build Trigger.

## Requirements
* These instructions assume that you are using a Bash Shell, and that you have [gcloud](https://cloud.google.com/sdk/install) installed and authenticated.
* You will need a domain name to use for the Cloud CDN Endpoint and for the **get-speech-service** URL.

## Deployment Instructions

### Setup Environment

1. Set variables:
```
PROJECT_ID=<PROJECT_ID>
GCS_TTS_BUCKET_NAME=<Bucket Name>
GCS_TTS_BUCKET_LOCATION=<Location>
TTS_CDN_FQDN=<FQDN>
CLOUD_RUN_REGION=<REGION>
```

<details><summary>Variable Explanation</summary>
<p>

| Variable                     | Description                                                                                                                                                                             |
| ---------------------------  |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------     |
| PROJECT_ID                   | The Google Cloud Project ID to deploy the resources in.                                                                                                                                 |
| GCS_TTS_BUCKET_NAME          | The GCS Bucket Name to create for storage of the TTS synthesised audio.                                                                                                                 |
| GCS_TTS_BUCKET_LOCATION      | The location to create the GCS Bucket in (see [Bucket locations](https://cloud.google.com/storage/docs/locations)) .                                                                    |
| TTS_CDN_FQDN                 | The FQDN to use for the CDN endpoint.                                                                                                                                                   |
| CLOUD_RUN_REGION             | The region to deploy the Cloud Run Services in (see: [Cloud Run locations](https://cloud.google.com/run/docs/locations)).                                                               |


</p>
</details>

2. Set gcloud project and additional variables:
```
glcoud config set project ${PROJECT_ID}
PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)")
```

### Enable Required Google API's
```
gcloud services enable storage-component.googleapis.com compute.googleapis.com texttospeech.googleapis.com secretmanager.googleapis.com cloudbuild.googleapis.com run.googleapis.com containerregistry.googleapis.com
```

### Build the get-speech-service Container and Store in Google Container Registry

Do not complete this step if setting up a local development environment

```
gcloud builds submit --tag gcr.io/${PROJECT_ID}/get-speech-service:latest .
```

### Create a Service Account for the get-speech-service
```
gcloud iam service-accounts create get-speech-service-sa  --description="Service Account for the get-speech-service"
```

### Create GCS Bucket and Grant SA Permissions
```
gsutil mb -c standard -l ${GCS_TTS_BUCKET_LOCATION} gs://${GCS_TTS_BUCKET_NAME}
gsutil iam ch serviceAccount:get-speech-service-sa@${PROJECT_ID}.iam.gserviceaccount.com:roles/storage.objectCreator gs://${GCS_TTS_BUCKET_NAME}
gsutil iam ch serviceAccount:get-speech-service-sa@${PROJECT_ID}.iam.gserviceaccount.com:roles/storage.objectViewer gs://${GCS_TTS_BUCKET_NAME}
```

### Setup Cloud CDN
1. Reserve an IPv4 Address:
```
gcloud compute addresses create get-speech-service-cdn-ip --network-tier=PREMIUM --ip-version=IPV4 --global
TTS_CDN_LB_IP=$(gcloud compute addresses describe get-speech-service-cdn-ip --format="get(address)" --global)
echo ${TTS_CDN_LB_IP}
```

2. Login to your DNS provider, and create a DNS record using the outputs from the below command:
```
echo "FQDN: ${TTS_CDN_FQDN}"
echo "A Record: ${TTS_CDN_LB_IP}"
```

3. Create Load Balancer Components
```
gcloud compute backend-buckets create get-speech-service-cdn-backend --gcs-bucket-name=${GCS_TTS_BUCKET_NAME} --description="Backend for the get-speech-service CDN" --enable-cdn
gcloud compute url-maps create get-speech-service-cdn-url-maps --default-backend-bucket=get-speech-service-cdn-backend --description="url-map for the get-speech-service CDN"
gcloud compute ssl-certificates create get-speech-service-cdn-ssl-certificate --description="SSL Certificate for the get-speech-service CDN" --domains="${TTS_CDN_FQDN}" --global
gcloud compute target-https-proxies create get-speech-service-cdn-https-proxy --ssl-certificates=get-speech-service-cdn-ssl-certificate --url-map=get-speech-service-cdn-url-maps
gcloud compute forwarding-rules create get-speech-service-cdn-forwarding-rule --address=get-speech-service-cdn-ip --target-https-proxy=get-speech-service-cdn-https-proxy --global --ports=443
```

### Generate Signed URL Key, and to Load Balancer Backend
```
head -c 16 /dev/urandom | base64 | tr +/ -_ > localsecrets/get-speech-service-signed-url-key.key
gcloud compute backend-buckets add-signed-url-key get-speech-service-cdn-backend --key-name get-speech-service-signed-url-key --key-file localsecrets/get-speech-service-signed-url-key.key
```

### Grant the Cloud CDN Service Account Read Permission to the GCS Bucket
```
gsutil iam ch serviceAccount:service-${PROJECT_NUMBER}@cloud-cdn-fill.iam.gserviceaccount.com:objectViewer gs://${GCS_TTS_BUCKET_NAME}
```

### Create a Secrets Manager Secret to store Signed URL Key
```
gcloud secrets create get-speech-service-cdn-signing-key --replication-policy=automatic
```

### Create a Secret Version from the Signed URL Key File
```
gcloud secrets versions add get-speech-service-cdn-signing-key --data-file=localsecrets/get-speech-service-signed-url-key.key
```

### Modify sample iam-policy.yml
1. Edit the file located in [resources/iam-policy.yml](resources/iam-policy.yml) and update the `members` mapping to contain the required Service Accounts. This can also contain any local development Service Accounts. Ensure that as a minimum the Service Account from the below command output is included in the members list:
```
echo "serviceAccount:get-speech-service-sa@${PROJECT_ID}.iam.gserviceaccount.com"
```

### Apply IAM Policy to get-speech-service-cdn-signing-key Secret
```
gcloud secrets set-iam-policy get-speech-service-cdn-signing-key resources/iam-policy.yml
```

### Deploy get-speech-service to Cloud Run

Do not complete this step if setting up a local development environment

1. Run the following command (tweak config as appropriate. See [gcloud run deploy docs](https://cloud.google.com/sdk/gcloud/reference/run/deploy))
```
gcloud run deploy get-speech-service-2 \
  --image=gcr.io/${PROJECT_ID}/get-speech-service:latest \
  --region=${CLOUD_RUN_REGION} \
  --platform=managed \
  --concurrency=80 \
  --cpu=2 \
  --memory=1Gi \
  --timeout=20s \
  --max-instances=100 \
  --service-account=get-speech-service-sa@$PROJECT_ID.iam.gserviceaccount.com \
  --set-env-vars=GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_PROJECT_NUMBER=${PROJECT_NUMBER},GCS_BUCKET_NAME=${GCS_TTS_BUCKET_NAME},CLOUD_CDN_SIGNING_KEY_SECRET_NAME=get-speech-service-cdn-signing-key,CLOUD_CDN_SIGNED_URL_KEY_NAME=get-speech-service-signed-url-key,CLOUD_CDN_ENDPOINT_FQDN=${TTS_CDN_FQDN} \
  --allow-unauthenticated
```

### Optional: Grant Cloud Build Permissions
1. Grant Cloud Build permissions to deploy to Cloud Run, and to use the required Service Accounts.
```
gcloud projects add-iam-policy-binding $PROJECT_ID --member serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com --role='roles/run.admin'
gcloud iam service-accounts add-iam-policy-binding $PROJECT_NUMBER-compute@developer.gserviceaccount.com  --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" --role='roles/iam.serviceAccountUser'
gcloud iam service-accounts add-iam-policy-binding get-speech-service-sa@$PROJECT_ID.iam.gserviceaccount.com  --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" --role='roles/iam.serviceAccountUser'
```

### Optional: Create Build Trigger
See [here](https://cloud.google.com/cloud-build/docs/automating-builds/create-manage-triggers)

### Optional: Create Custom Domain Mapping for get-speech-service
1. Set Environment
```
GET_SPEECH_SERVICE_FQDN=<FQDN>
```
<details><summary>Variable Explanation</summary>
<p>

| Variable                     | Description                                                                                                                                                                            |
| ---------------------------  |--------------------------------------------------------------------------------------------------------------                                                                          |
| GET_SPEECH_SERVICE_FQDN      | The custom URL you want to use for the get-speech-service Cloud Run Service (e.g. `get-speech-service.demos.maynard.io`)                                                               |

</p>
</details>

2. Create Domain Mapping
```
gcloud beta run domain-mappings create --service get-speech-service --domain ${GET_SPEECH_SERVICE_FQDN} --platform=managed --region=${CLOUD_RUN_REGION}
```

3. Login to your DNS provider, and create a DNS record as described in the above command output

4. Monitor the status of the Domain Mapping
```
gcloud beta run domain-mappings list --platform=managed --region=${CLOUD_RUN_REGION}
```

# Developing Locally

Developing locally requires [docker-compose](https://docs.docker.com/compose/install/).

A [docker-compose.yml](docker-compose.yml) file is included in this directory to simplify the configuration when testing locally. You should create a separate development project in Google Cloud and follow all of the steps above, with the exception of deploying the image to Cloud Run. Once complete, perform the following steps for local development.

### Generate Key for the Service Account
```
mkdir -p localsecrets
gcloud iam service-accounts keys create localsecrets/service-account.json --iam-account get-speech-service-sa@$PROJECT_ID.iam.gserviceaccount.com
```

### Setup Local Environment
```
echo "PORT=80" >> local.env
echo "GOOGLE_APPLICATION_CREDENTIALS=/tmp/service-account.json" >> local.env
echo "GOOGLE_CLOUD_PROJECT=${PROJECT_ID}" >> local.env
echo "GOOGLE_CLOUD_PROJECT_NUMBER=${PROJECT_NUMBER}"  >> local.env
echo "GCS_BUCKET_NAME=${GCS_TTS_BUCKET_NAME}" >> local.env
echo "CLOUD_CDN_SIGNING_KEY_SECRET_NAME=get-speech-service-cdn-signing-key" >> local.env
echo "CLOUD_CDN_SIGNED_URL_KEY_NAME=get-speech-service-signed-url-key" >> local.env
echo "CLOUD_CDN_ENDPOINT_FQDN=${TTS_CDN_FQDN}" >> local.env
```


### Build and Run Container
```
docker-compose build && docker-compose up -d
```

### Check Container Status
```
docker-compose ps
```

### Access Local Environment
1. Visit https://localhost:8008
