#!/bin/bash
set -e

# Define log file for debugging
LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/greenfield.log"
mkdir -p "$(dirname "$LOG_FILE")"  # Ensure the log directory exists

CONTAINER_NAME=s3_and_iam_deployment-iam-1
HOST_IP=$(hostname -I | awk '{print $1}')
BUCKET_CLASS_NAME="my-bucket-class"
IAM_ENDPOINT="http://$HOST_IP:8600"
S3_ENDPOINT="http://$HOST_IP:8000"
ACCOUNT_ACCESS_KEY_ID="PBUOB68AVF39EVVAFNFL"
ACCOUNT_SECRET_ACCESS_KEY="P+PK+uMB9spUc21huaQoOexqdJoV00tSnl+pc7t7"
REGION="us-west-1"
IAM_USER_PREFIX="ba"
SECRET_NAME="object-storage-access-secret"
NAMESPACE="default"
ADMIN_ACCESS_KEY_ID=D4IT2AWSB588GO5J9T00
ADMIN_SECRET_ACCESS_KEY=UEEu8tYlsOGGrgf4DAiSZD6apVNPUWqRiPG0nTB6
ATTEMPTS=12  # Total AWS CLI attempts (2 minutes / 10 seconds per attempt)
DELAY=10  # Delay between AWS CLI requests attempts in seconds


# Error handling function
error_handler() {
  echo "An error occurred during bucket creation tests. Check the log file for details." | tee -a "$LOG_FILE"
  echo "Failed command: $BASH_COMMAND" | tee -a "$LOG_FILE"
  exit 1
}

# Trap errors and call the error handler
trap 'error_handler' ERR

log_and_run() {
  echo "Running: $*" | tee -a "$LOG_FILE"
  if ! "$@" 2>&1 | tee -a "$LOG_FILE"; then
    echo "Error: Command failed - $*" | tee -a "$LOG_FILE"
    exit 1
  fi
}

# Step 1: Create Account in Vault
log_and_run echo "Creating account in Vault container..."
log_and_run docker exec "$CONTAINER_NAME" sh -c "ADMIN_ACCESS_KEY_ID=$ADMIN_ACCESS_KEY_ID ADMIN_SECRET_ACCESS_KEY=$ADMIN_SECRET_ACCESS_KEY ./node_modules/vaultclient/bin/vaultclient create-account --name cosi-account --email cosi-account@scality.local"
log_and_run docker exec "$CONTAINER_NAME" sh -c "ADMIN_ACCESS_KEY_ID=$ADMIN_ACCESS_KEY_ID ADMIN_SECRET_ACCESS_KEY=$ADMIN_SECRET_ACCESS_KEY ./node_modules/vaultclient/bin/vaultclient generate-account-access-key --name=cosi-account --accesskey=$ACCOUNT_ACCESS_KEY_ID --secretkey=$ACCOUNT_SECRET_ACCESS_KEY"

# Retrieve the Host IP Address
log_and_run echo "Using Host IP: $HOST_IP"

# Step 2: Configure AWS CLI in Home Directory
log_and_run echo "Configuring AWS CLI in home directory..."
log_and_run mkdir -p ~/.aws  # Ensure the ~/.aws directory exists

# Create the AWS credentials file
cat <<EOF | tee -a "$LOG_FILE" > ~/.aws/credentials
[default]
aws_access_key_id = $ACCOUNT_ACCESS_KEY_ID
aws_secret_access_key = $ACCOUNT_SECRET_ACCESS_KEY
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
  accessKeyId: $ACCOUNT_ACCESS_KEY_ID
  secretAccessKey: $ACCOUNT_SECRET_ACCESS_KEY
  endpoint: $S3_ENDPOINT
  region: $REGION
  iamEndpoint: $IAM_ENDPOINT 
EOF

# Step 4: Apply Bucket Class
log_and_run echo "Applying Bucket Class..."
log_and_run kubectl apply -f cosi-examples/greenfield/bucketclass.yaml

# Step 5: Apply Bucket Claim
log_and_run echo "Applying Bucket Claim..."
log_and_run kubectl apply -f cosi-examples/greenfield/bucketclaim.yaml

# Step 6: Apply Bucket Access Class
log_and_run echo "Applying Bucket Access Class..."
log_and_run kubectl apply -f cosi-examples/greenfield/bucketaccessclass.yaml

# Step 7: Apply Bucket Access
log_and_run echo "Applying Bucket Access..."
log_and_run kubectl apply -f cosi-examples/greenfield/bucketaccess.yaml

# Step 8: Verify Bucket Creation with Retry
log_and_run echo "Listing all S3 buckets before verification..."
log_and_run aws s3 ls --endpoint-url "$S3_ENDPOINT"
sleep 5

log_and_run echo "Verifying bucket creation..."

for ((i=1; i<=$ATTEMPTS; i++)); do
  log_and_run aws --endpoint-url "$S3_ENDPOINT" s3 ls
  BUCKET_FOUND=$(aws --endpoint-url "$S3_ENDPOINT" s3api list-buckets --query "Buckets[?starts_with(Name, '$BUCKET_CLASS_NAME')].Name" --output text)

  if [ -n "$BUCKET_FOUND" ]; then
    log_and_run echo "Bucket created with prefix '$BUCKET_CLASS_NAME': $BUCKET_FOUND"
    break
  else
    log_and_run echo "Attempt $i: Bucket with prefix '$BUCKET_CLASS_NAME' not found. Retrying in $DELAY seconds..."
    sleep $DELAY
  fi
done

if [ -z "$BUCKET_FOUND" ]; then
  log_and_run echo "Bucket with prefix '$BUCKET_CLASS_NAME' was not created."
  exit 1
fi

# Step 9: Verify IAM User and Access Key Creation
log_and_run echo "Verifying IAM user with prefix '$IAM_USER_PREFIX'..."
log_and_run aws --endpoint-url "$IAM_ENDPOINT" iam list-users

IAM_USER_NAME=$(aws --endpoint-url "$IAM_ENDPOINT" iam list-users --query "Users[?starts_with(UserName, '$IAM_USER_PREFIX')].UserName" --output text)
if [ -z "$IAM_USER_NAME" ]; then
  log_and_run echo "IAM user with prefix '$IAM_USER_PREFIX' not found."
  exit 1
fi
log_and_run echo "IAM user found: $IAM_USER_NAME"

log_and_run echo "Verifying inline policy attached to IAM user..."
INLINE_POLICY="$(aws --endpoint-url "$IAM_ENDPOINT" iam list-user-policies --user-name "$IAM_USER_NAME" --query "PolicyNames[0]" --output text)"
EXPECTED_INLINE_POLICY="$BUCKET_FOUND"

if [[ "$INLINE_POLICY" != "$EXPECTED_INLINE_POLICY" ]]; then
  log_and_run echo "Inline policy '$INLINE_POLICY' does not match expected bucket name '$EXPECTED_INLINE_POLICY'."
  exit 1
fi
log_and_run echo "Inline policy '$INLINE_POLICY' matches bucket name '$EXPECTED_INLINE_POLICY'."

log_and_run echo "Verifying access key for IAM user..."
IAM_USER_ACCESS_KEY="$(aws --endpoint-url "$IAM_ENDPOINT" iam list-access-keys --user-name "$IAM_USER_NAME" --query "AccessKeyMetadata[0].AccessKeyId" --output text)"

if [ -z "$IAM_USER_ACCESS_KEY" ]; then
  log_and_run echo "Access key not found for IAM user '$IAM_USER_NAME'."
  exit 1
fi
log_and_run echo "Access key found for IAM user: $IAM_USER_ACCESS_KEY"


# Step 10: Verify the object-storage-access-secret
log_and_run echo "Verifying object-storage-access-secret in the default namespace..."

# Fetch the secret data as JSON
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
if [[ "$ACTUAL_BUCKET_NAME" != "$BUCKET_FOUND" ]]; then
  log_and_run echo "Bucket name mismatch! Expected: $BUCKET_FOUND, Found: $ACTUAL_BUCKET_NAME"
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

# Verify accessKeyID
if [[ "$ACTUAL_ACCESS_KEY_ID" != "$IAM_USER_ACCESS_KEY" ]]; then
  log_and_run echo "AccessKeyID mismatch! Expected: $IAM_USER_ACCESS_KEY, Found: $ACTUAL_ACCESS_KEY_ID"
  exit 1
fi

# Verify accessSecretKey exists
if [[ -z "$ACTUAL_ACCESS_SECRET_KEY" ]]; then
  log_and_run echo "AccessSecretKey is empty!"
  exit 1
fi

# Verify protocols
EXPECTED_PROTOCOLS='["s3"]'
if [[ "$ACTUAL_PROTOCOLS" != "$EXPECTED_PROTOCOLS" ]]; then
  log_and_run echo "Protocols mismatch! Expected: $EXPECTED_PROTOCOLS, Found: $ACTUAL_PROTOCOLS"
  exit 1
fi

# Step 11: Delete Bucket Access Resource
log_and_run echo "Deleting Bucket Access Resource..."
log_and_run kubectl delete -f cosi-examples/greenfield/bucketaccess.yaml

# Step 12: Verify IAM User Deletion
log_and_run echo "Verifying IAM user '$IAM_USER_NAME' deletion..."
log_and_run aws --endpoint-url "$IAM_ENDPOINT" iam get-user --user-name "$IAM_USER_NAME"

USER_EXISTS="$(aws --endpoint-url "$IAM_ENDPOINT" iam get-user --user-name "$IAM_USER_NAME" --retry-mode standard --max-attempts 12 --delay $DELAY 2>&1 || true)"

if [[ "$USER_EXISTS" == *"NoSuchEntity"* ]]; then
  log_and_run echo "IAM user '$IAM_USER_NAME' successfully deleted."
else
  log_and_run echo "IAM user '$IAM_USER_NAME' still exists after retries."
fi

# Step 13: Test deletion bucket with deletion policy set

log_and_run echo "Applying Bucket Class with deletion policy and respective Bucket Claim..."
log_and_run kubectl apply -f cosi-examples/greenfield/bucketclass-deletion-policy.yaml
log_and_run kubectl apply -f cosi-examples/greenfield/bucketclaim-deletion-policy.yaml

log_and_run echo "Listing all S3 buckets before deletion..."
log_and_run aws s3 ls --endpoint-url "$S3_ENDPOINT"

BUCKET_CLASS_NAME="delete-bucket-class"

log_and_run echo "Verifying bucket creation with prefix '$BUCKET_CLASS_NAME'..."

for ((i=1; i<=$ATTEMPTS; i++)); do
  log_and_run aws --endpoint-url "$S3_ENDPOINT" s3 ls
  BUCKET_TO_BE_DELETED=$(aws --endpoint-url "$S3_ENDPOINT" s3api list-buckets --query "Buckets[?starts_with(Name, '$BUCKET_CLASS_NAME')].Name" --output text)

  if [ -n "$BUCKET_TO_BE_DELETED" ]; then
    log_and_run echo "Bucket created with prefix '$BUCKET_CLASS_NAME': $BUCKET_TO_BE_DELETED"
    break
  else
    log_and_run echo "Attempt $i: Bucket with prefix '$BUCKET_CLASS_NAME' not found. Retrying in $DELAY seconds..."
    sleep $DELAY
  fi
done

if [ -z "$BUCKET_TO_BE_DELETED" ]; then
  log_and_run echo "Bucket with prefix '$BUCKET_CLASS_NAME' was not created."
  exit 1
fi

log_and_run echo "Deleting Bucket Claim..."
log_and_run kubectl delete -f cosi-examples/greenfield/bucketclaim-deletion-policy.yaml

# Check if the bucket with name $BUCKET_TO_BE_DELETED exists by doing a head bucket.
# If bucket exists, retry with ATTEMPTS and DELAY. If bucket is not found, test success.

log_and_run echo "Verifying bucket deletion with name '$BUCKET_TO_BE_DELETED'..."

# Check if the bucket has been deleted
log_and_run aws s3 ls --endpoint-url "$S3_ENDPOINT"
AWS_MAX_ATTEMPTS=$ATTEMPTS
AWS_RETRY_DELAY=$DELAY
# Run head-bucket to check if the bucket has been deleted
BUCKET_HEAD_RESULT="$(log_and_run aws --endpoint-url "$S3_ENDPOINT" s3api head-bucket --bucket "$BUCKET_TO_BE_DELETED" 2>&1 || true)"

# Log the actual error result for debugging purposes
log_and_run echo "head-bucket result: $BUCKET_HEAD_RESULT"

# Check if the result contains the "Not Found" error message
if [[ "$BUCKET_HEAD_RESULT" == *"Not Found"* ]]; then
  log_and_run echo "Bucket with name '$BUCKET_TO_BE_DELETED' was successfully deleted (Not Found error)."
else
  log_and_run echo "Bucket with name '$BUCKET_TO_BE_DELETED' was not deleted after $ATTEMPTS attempts. Error: $BUCKET_HEAD_RESULT"
  exit 1
fi
