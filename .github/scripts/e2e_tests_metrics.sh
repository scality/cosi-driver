#!/bin/bash
set -e

LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/metrics_service.log"
mkdir -p "$(dirname "$LOG_FILE")"

NAMESPACE="scality-object-storage"
SERVICE="scality-cosi-driver-metrics"
LOCAL_PORT=8080
TARGET_PORT=8080

EXPECTED_CREATE_BUCKET=2
EXPECTED_DELETE_BUCKET=1
EXPECTED_GET_INFO=1
EXPECTED_GRANT_ACCESS=2
EXPECTED_REVOKE_ACCESS=2
GRPC_METHOD_TO_TEST="grpc_server_msg_sent_total"

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
  "$@" 2>&1 | tee -a "$LOG_FILE"
}

log_and_run kubectl get svc --all-namespaces

echo "Starting port-forwarding for service $SERVICE in namespace $NAMESPACE" | tee -a "$LOG_FILE"
kubectl port-forward -n "$NAMESPACE" svc/"$SERVICE" "$LOCAL_PORT":"$TARGET_PORT" > /dev/null 2>&1 &
PORT_FORWARD_PID=$!

# Wait a few seconds to ensure port-forward is established
sleep 5

# Fetch metrics
echo "Fetching metrics from localhost:$LOCAL_PORT/metrics" | tee -a "$LOG_FILE"
METRICS_OUTPUT=$(curl -s http://localhost:$LOCAL_PORT/metrics | grep $GRPC_METHOD_TO_TEST)
echo "Metrics fetched successfully:" | tee -a "$LOG_FILE"
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
    kill "$PORT_FORWARD_PID" # Clean up port-forwarding before exiting
    exit 1
  fi
done

echo "All metrics validated successfully." | tee -a "$LOG_FILE"

# Clean up port-forwarding
echo "Cleaning up port-forwarding (PID: $PORT_FORWARD_PID)" | tee -a "$LOG_FILE"
kill "$PORT_FORWARD_PID"

# Echo the metrics environment variable for reference
export GRPC_METRICS="$METRICS_OUTPUT"
echo "GRPC_METRICS: $GRPC_METRICS" | tee -a "$LOG_FILE"