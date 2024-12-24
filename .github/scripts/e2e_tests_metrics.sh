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
    log_and_run kill "$PORT_FORWARD_PID" # Clean up port-forwarding before exiting
    exit 1
  fi
done

log_and_run echo "Verifying S3 and IAM metrics..."
# only verify metrics if EXPECTED_CREATE_BUCKET is more than 0

if [[ "$EXPECTED_CREATE_BUCKET" -gt 0 ]]; then
  S3_IAM_METRICS_OUTPUT=$(cat  /tmp/metrics_output.log | grep 'scality_cosi_driver')
  echo "Metrics fetched successfully:" | tee -a "$LOG_FILE"
  echo "$S3_IAM_METRICS_OUTPUT" | tee -a "$LOG_FILE"

  log_and_run cat /tmp/s3_iam_metrics_output.log
fi

echo "Metrics validation successful!" | tee -a "$LOG_FILE"