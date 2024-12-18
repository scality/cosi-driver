#!/bin/bash
set -e

LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/brownfield.log"
mkdir -p "$(dirname "$LOG_FILE")"

HOST_IP=$(hostname -I | awk '{print $1}')
SECRET_NAME="brownfield-bucket-secret"
IAM_ENDPOINT="http://$HOST_IP:8600"
S3_ENDPOINT="http://$HOST_IP:8000"
BUCKET_NAME="brownfield-bucket"
NAMESPACE="scality-object-storage"
REGION="us-west-1"

# Error handling function
error_handler() {
  echo "An error occurred during bucket creation tests. Check the log file for details." | tee -a "$LOG_FILE"
  echo "Failed command: $BASH_COMMAND" | tee -a "$LOG_FILE"
  exit 1
}

# Trap errors and call the error handler
trap 'error_handler' ERR

# Log command execution to the log file for debugging
log_and_run() {
  "$@" 2>&1 | tee -a "$LOG_FILE"
}


# Create the bucket fir brownfield scenario
log_and_run echo "Creating bucket: $BUCKET_NAME"
log_and_run aws s3api create-bucket --bucket "$BUCKET_NAME" --region $REGION --endpoint-url "$S3_ENDPOINT"

# Check if the bucket exists
log_and_run echo "Checking if bucket $BUCKET_NAME exists"
aws --endpoint-url "$S3_ENDPOINT" s3api head-bucket --bucket "$BUCKET_NAME"
log_and_run echo "Bucket $BUCKET_NAME exists!"

log_and_run echo "Applying Bucket Class to use existing bucket..."
log_and_run kubectl apply -f cosi-examples/brownfield/bucketclass.yaml

log_and_run echo "Manually creating Bucket object with existing bucket..."
log_and_run kubectl apply -f cosi-examples/brownfield/bucket.yaml

log_and_run echo "Applying Bucket Claim referencing the Bucket object..."
log_and_run kubectl apply -f cosi-examples/brownfield/bucketclaim.yaml

log_and_run echo "Applying Bucket Access Class..."
log_and_run kubectl apply -f cosi-examples/brownfield/bucketaccessclass.yaml

log_and_run echo "Applying Bucket Access..."
log_and_run kubectl apply -f cosi-examples/brownfield/bucketaccess.yaml

log_and_run echo "Verifying brownfield-bucket-secret in the default namespace..."
SECRET_JSON="$(kubectl get secret "$SECRET_NAME" --namespace "$NAMESPACE" -o json)"

# Decode the Base64 encoded BucketInfo
BUCKET_INFO_BASE64="$(echo "$SECRET_JSON" | jq -r '.data.BucketInfo')"
BUCKET_INFO_JSON="$(echo "$BUCKET_INFO_BASE64" | base64 --decode)"

log_and_run echo "Decoded BucketInfo: $BUCKET_INFO_JSON"

# Extract values to verify
ACTUAL_BUCKET_NAME=$(echo "$BUCKET_INFO_JSON" | jq -r '.spec.bucketName')
ACTUAL_ENDPOINT=$(echo "$BUCKET_INFO_JSON" | jq -r '.spec.secretS3.endpoint')
ACTUAL_REGION=$(echo "$BUCKET_INFO_JSON" | jq -r '.spec.secretS3.region')
ACTUAL_ACCESS_KEY_ID=$(echo "$BUCKET_INFO_JSON" | jq -r '.spec.secretS3.accessKeyID')
ACTUAL_ACCESS_SECRET_KEY=$(echo "$BUCKET_INFO_JSON" | jq -r '.spec.secretS3.accessSecretKey')
ACTUAL_PROTOCOLS=$(echo "$BUCKET_INFO_JSON" | jq -c '.spec.protocols')

# Verify bucketName
if [[ "$ACTUAL_BUCKET_NAME" != "$BUCKET_NAME" ]]; then
  log_and_run echo "Bucket name mismatch! Expected: $BUCKET_NAME, Found: $ACTUAL_BUCKET_NAME"
  exit 1
fi

# Verify endpoint
EXPECTED_ENDPOINT="$S3_ENDPOINT"
if [[ "$ACTUAL_ENDPOINT" != "$EXPECTED_ENDPOINT" ]]; then
  log_and_run echo "Endpoint mismatch! Expected: $EXPECTED_ENDPOINT, Found: $ACTUAL_ENDPOINT"
  exit 1
fi

# Verify region
if [[ "$ACTUAL_REGION" != "$REGION" ]]; then
  log_and_run echo "Region mismatch! Expected: $REGION, Found: $ACTUAL_REGION"
  exit 1
fi

# Verify accessSecretKey exists
if [[ -z "$ACTUAL_ACCESS_KEY_ID" ]]; then
  log_and_run echo "AccessSecretKey is empty!"
  exit 1
fi

# Verify accessSecretKey exists
if [[ -z "$ACTUAL_ACCESS_SECRET_KEY" ]]; then
  log_and_run echo "AccessSecretKey is empty!"
  exit 1
fi

# Verify protocol
EXPECTED_PROTOCOLS='["s3"]'
if [[ "$ACTUAL_PROTOCOLS" != "$EXPECTED_PROTOCOLS" ]]; then
  log_and_run echo "Protocols mismatch! Expected: $EXPECTED_PROTOCOLS, Found: $ACTUAL_PROTOCOLS"
  exit 1
fi

# cleanup
log_and_run kubectl delete -f cosi-examples/brownfield/bucketaccess.yaml
log_and_run kubectl delete -f cosi-examples/brownfield/bucketaccessclass.yaml
log_and_run kubectl delete -f cosi-examples/brownfield/bucketclaim.yaml
log_and_run kubectl delete -f cosi-examples/brownfield/bucketclass.yaml

# Check if the bucket is not deleted and Retain policy is respected
log_and_run echo "Checking if bucket $BUCKET_NAME exists"
aws --endpoint-url "$S3_ENDPOINT" s3api head-bucket --bucket "$BUCKET_NAME"
log_and_run echo "Bucket $BUCKET_NAME has been retained!"
