#!/bin/bash
set -e

# Define log file for debugging
LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/bucket_creation_test.log"
mkdir -p "$(dirname "$LOG_FILE")"  # Ensure the log directory exists

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
  echo "Running: $*" | tee -a "$LOG_FILE"
  "$@" | tee -a "$LOG_FILE"
}

# Step 1: Create Account in Vault
log_and_run echo "Creating account in Vault container..."
CONTAINER_NAME=s3_and_iam_deployment-iam-1
log_and_run docker exec "$CONTAINER_NAME" sh -c "ADMIN_ACCESS_KEY_ID=D4IT2AWSB588GO5J9T00 ADMIN_SECRET_ACCESS_KEY=UEEu8tYlsOGGrgf4DAiSZD6apVNPUWqRiPG0nTB6 ./node_modules/vaultclient/bin/vaultclient create-account --name cosi-account --email cosi-account@scality.local"
log_and_run docker exec "$CONTAINER_NAME" sh -c "ADMIN_ACCESS_KEY_ID=D4IT2AWSB588GO5J9T00 ADMIN_SECRET_ACCESS_KEY=UEEu8tYlsOGGrgf4DAiSZD6apVNPUWqRiPG0nTB6 ./node_modules/vaultclient/bin/vaultclient generate-account-access-key --name=cosi-account --accesskey=PBUOB68AVF39EVVAFNFL --secretkey=P+PK+uMB9spUc21huaQoOexqdJoV00tSnl+pc7t7"

# Retrieve the Host IP Address
HOST_IP=$(hostname -I | awk '{print $1}')
log_and_run echo "Using Host IP: $HOST_IP"

# Step 2: Configure AWS CLI in Home Directory
log_and_run echo "Configuring AWS CLI in home directory..."
log_and_run mkdir -p ~/.aws  # Ensure the ~/.aws directory exists

# Create the AWS credentials file
cat <<EOF | tee -a "$LOG_FILE" > ~/.aws/credentials
[default]
aws_access_key_id = PBUOB68AVF39EVVAFNFL
aws_secret_access_key = P+PK+uMB9spUc21huaQoOexqdJoV00tSnl+pc7t7
EOF

# Create the AWS config file
cat <<EOF | tee -a "$LOG_FILE" > ~/.aws/config
[default]
region = us-east-1
output = json
EOF

# Step 3: Apply S3 Secret for COSI with Host IP
log_and_run echo "Applying S3 Secret for COSI with updated endpoint..."
cat <<EOF | kubectl apply -f - | tee -a "$LOG_FILE"
apiVersion: v1
kind: Secret
metadata:
  name: s3-secret-for-cosi
  namespace: default
type: Opaque
stringData:
  accessKeyId: PBUOB68AVF39EVVAFNFL
  secretAccessKey: P+PK+uMB9spUc21huaQoOexqdJoV00tSnl+pc7t7
  endpoint: http://$HOST_IP:8000
  region: us-west-1
EOF

# Step 4: Apply Bucket Class
log_and_run echo "Applying Bucket Class..."
log_and_run kubectl apply -f cosi-examples/bucketclass.yaml

# Step 5: Apply Bucket Claim
log_and_run echo "Applying Bucket Claim..."
log_and_run kubectl apply -f cosi-examples/bucketclaim.yaml

# Step 6: Verify Bucket Creation with Retry
log_and_run echo "Listing all S3 buckets before verification..."
log_and_run aws s3 ls --endpoint-url "http://localhost:8000"
sleep 5

log_and_run echo "Verifying bucket creation..."
BUCKET_NAME_PREFIX="my-bucket-class"

ATTEMPTS=12  # Total attempts (2 minutes / 10 seconds per attempt)
DELAY=10  # Delay between attempts in seconds

for ((i=1; i<=$ATTEMPTS; i++)); do
  log_and_run aws --endpoint-url "http://localhost:8000" s3 ls
  BUCKET_FOUND=$(aws --endpoint-url "http://localhost:8000" s3api list-buckets --query "Buckets[?starts_with(Name, 'my-bucket-class')].Name" --output text)

  if [ -n "$BUCKET_FOUND" ]; then
    log_and_run echo "Bucket created with prefix '$BUCKET_NAME_PREFIX': $BUCKET_FOUND"
    exit 0
  else
    log_and_run echo "Attempt $i: Bucket with prefix '$BUCKET_NAME_PREFIX' not found. Retrying in $DELAY seconds..."
    sleep $DELAY
  fi
done

# If the bucket was not found within the timeout
log_and_run echo "Bucket with prefix '$BUCKET_NAME_PREFIX' was not created within the expected time."
exit 1
