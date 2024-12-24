#!/bin/bash
set -e

LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/metrics_service.log"
mkdir -p "$(dirname "$LOG_FILE")"

NAMESPACE="scality-object-storage"
SERVICE="scality-cosi-driver-metrics"
LOCAL_PORT=8080
TARGET_PORT=8080

# Declare expected values for each metric as environment variables
EXPECTED_CREATE_BUCKET=${1:-0}
EXPECTED_DELETE_BUCKET=${2:-0}
EXPECTED_GET_INFO=${3:-1}
EXPECTED_GRANT_ACCESS=${4:-0}
EXPECTED_REVOKE_ACCESS=${5:-0}
GRPC_METHOD_TO_TEST="grpc_server_msg_sent_total"

# Error handling function
error_handler() {
  echo "An error occurred during the metrics test. Check the log file for details." | tee -a "$LOG_FILE"
  echo "Failed command: $BASH_COMMAND" | tee -a "$LOG_FILE"
  exit 1
}

# Trap errors and call the error handler
trap 'error_handler' ERR

# Logging and command execution function
log_and_run() {
  echo "Running: $*" | tee -a "$LOG_FILE"
  "$@" 2>&1 | tee -a "$LOG_FILE"
}

# Fetch services and validate the target service exists
log_and_run kubectl get svc --all-namespaces

# Port-forward the metrics service
log_and_run kubectl port-forward -n "$NAMESPACE" svc/"$SERVICE" "$LOCAL_PORT":"$TARGET_PORT" &
PORT_FORWARD_PID=$!

# Wait a few seconds to ensure port-forward is established
log_and_run sleep 5

# Fetch metrics
log_and_run curl -s http://localhost:$LOCAL_PORT/metrics > /tmp/metrics_output.log
log_and_run cat /tmp/metrics_output.log

log_and_run kill "$PORT_FORWARD_PID"


METRICS_OUTPUT=$(cat /tmp/metrics_output.log | grep $GRPC_METHOD_TO_TEST)
echo "gRPC Metrics fetched successfully:" | tee -a "$LOG_FILE"
echo "$METRICS_OUTPUT" | tee -a "$LOG_FILE"

# Validate metrics
echo "Validating gRPC Server Metrics..." | tee -a "$LOG_FILE"
echo "$METRICS_OUTPUT" | while read -r line; do
  # Extract the grpc_method and value
  method=$(echo "$line" | sed -n 's/.*grpc_method="\([^"]*\)".*/\1/p') # Extract method name
  value=$(echo "$line" | awk '{print $NF}')                           # Extract value

  # Determine the expected value based on the grpc_method
  case "$method" in
    "DriverCreateBucket")
      expected_value=$EXPECTED_CREATE_BUCKET
      ;;
    "DriverDeleteBucket")
      expected_value=$EXPECTED_DELETE_BUCKET
      ;;
    "DriverGetInfo")
      expected_value=$EXPECTED_GET_INFO
      ;;
    "DriverGrantBucketAccess")
      expected_value=$EXPECTED_GRANT_ACCESS
      ;;
    "DriverRevokeBucketAccess")
      expected_value=$EXPECTED_REVOKE_ACCESS
      ;;
    *)
      echo "Unknown method: $method. Skipping validation." | tee -a "$LOG_FILE"
      continue
      ;;
  esac

  # Display method, value, and expected value
  echo "Method: $method, Value: $value, Expected: $expected_value" | tee -a "$LOG_FILE"

  # Perform validation
  if [[ "$value" -ne "$expected_value" ]]; then
    echo "Error: $method has an unexpected value ($value). Expected: $expected_value" | tee -a "$LOG_FILE"
    exit 1
  fi
done

log_and_run echo "Verifying S3 and IAM metrics..."
# only verify metrics if EXPECTED_CREATE_BUCKET is more than 0

if [[ "$EXPECTED_CREATE_BUCKET" -gt 0 ]]; then

  S3_IAM_METRICS_OUTPUT=$(cat  /tmp/metrics_output.log | grep 'scality_cosi_driver')
  echo "Metrics fetched successfully:" | tee -a "$LOG_FILE"
  echo "$S3_IAM_METRICS_OUTPUT" | tee -a "$LOG_FILE"
  CREATE_BUCKET_COUNT="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_s3_requests_total' | grep 'method="CreateBucket"' | grep 'status="success"' | awk '{print $NF}')"
  DELETE_BUCKET_COUNT="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_s3_requests_total' | grep 'method="DeleteBucket"' | grep 'status="success"' | awk '{print $NF}')"
  CREATE_USER_COUNT="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_iam_requests_total' | grep 'method="CreateUser"' | grep 'status="success"' | awk '{print $NF}')"
  DELETE_USER_COUNT="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_iam_requests_total' | grep 'method="DeleteUser"' | grep 'status="success"' | awk '{print $NF}')"

  CREATE_BUCKET_DURATION="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_s3_request_duration_seconds_sum' | grep 'method="CreateBucket"' | awk '{print $NF}')"
  DELETE_BUCKET_DURATION="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_s3_request_duration_seconds_sum' | grep 'method="DeleteBucket"' | awk '{print $NF}')"
  CREATE_USER_DURATION="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_iam_request_duration_seconds_sum' | grep 'method="CreateUser"' | awk '{print $NF}')"
  DELETE_USER_DURATION="$(echo "$S3_IAM_METRICS_OUTPUT" | grep 'scality_cosi_driver_iam_request_duration_seconds_sum' | grep 'method="DeleteUser"' | awk '{print $NF}')"

  echo "CreateBucket Count: $CREATE_BUCKET_COUNT, Expected: $EXPECTED_CREATE_BUCKET" | tee -a "$LOG_FILE"
  echo "DeleteBucket Count: $DELETE_BUCKET_COUNT, Expected: $EXPECTED_DELETE_BUCKET" | tee -a "$LOG_FILE"
  echo "CreateUser Count: $CREATE_USER_COUNT, Expected: $EXPECTED_GRANT_ACCESS" | tee -a "$LOG_FILE"
  echo "DeleteUser Count: $DELETE_USER_COUNT, Expected: $EXPECTED_REVOKE_ACCESS" | tee -a "$LOG_FILE"

  echo "CreateBucket Duration: $CREATE_BUCKET_DURATION" | tee -a "$LOG_FILE"
  echo "DeleteBucket Duration: $DELETE_BUCKET_DURATION" | tee -a "$LOG_FILE"
  echo "CreateUser Duration: $CREATE_USER_DURATION" | tee -a "$LOG_FILE"
  echo "DeleteUser Duration: $DELETE_USER_DURATION" | tee -a "$LOG_FILE"

  # Validate counts
  if [[ "$CREATE_BUCKET_COUNT" -ne "$EXPECTED_CREATE_BUCKET" ]]; then
    echo "Error: CreateBucket count mismatch. Found: $CREATE_BUCKET_COUNT, Expected: $EXPECTED_CREATE_BUCKET" | tee -a "$LOG_FILE"
    exit 1
  fi
  if [[ "$DELETE_BUCKET_COUNT" -ne "$EXPECTED_DELETE_BUCKET" ]]; then
    echo "Error: DeleteBucket count mismatch. Found: $DELETE_BUCKET_COUNT, Expected: $EXPECTED_DELETE_BUCKET" | tee -a "$LOG_FILE"
    exit 1
  fi
  if [[ "$CREATE_USER_COUNT" -ne "$EXPECTED_GRANT_ACCESS" ]]; then
    echo "Error: CreateUser count mismatch. Found: $CREATE_USER_COUNT, Expected: $EXPECTED_GRANT_ACCESS" | tee -a "$LOG_FILE"
    exit 1
  fi

  if [[ "$DELETE_USER_COUNT" -ne "$EXPECTED_REVOKE_ACCESS" ]]; then
    echo "Error: DeleteUser count mismatch. Found: $DELETE_USER_COUNT, Expected: $EXPECTED_REVOKE_ACCESS" | tee -a "$LOG_FILE"
    exit 1
  fi

  # Validate durations are greater than 0
  if (( $(echo "$CREATE_BUCKET_DURATION <= 0" | bc -l) )); then
    echo "Error: CreateBucket duration is not greater than 0. Duration: $CREATE_BUCKET_DURATION" | tee -a "$LOG_FILE"
    exit 1
  fi
  if (( $(echo "$DELETE_BUCKET_DURATION <= 0" | bc -l) )); then
    echo "Error: DeleteBucket duration is not greater than 0. Duration: $DELETE_BUCKET_DURATION" | tee -a "$LOG_FILE"
    exit 1
  fi

  if (( $(echo "$CREATE_USER_DURATION <= 0" | bc -l) )); then
    echo "Error: CreateUser duration is not greater than 0. Duration: $CREATE_USER_DURATION" | tee -a "$LOG_FILE"
    exit 1
  fi

  if (( $(echo "$DELETE_USER_DURATION <= 0" | bc -l) )); then
    echo "Error: DeleteUser duration is not greater than 0. Duration: $DELETE_USER_DURATION" | tee -a "$LOG_FILE"
    exit 1
  fi
fi

echo "Metrics validation successful!" | tee -a "$LOG_FILE"
