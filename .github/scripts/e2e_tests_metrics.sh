#!/bin/bash
set -e

LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/metrics_service.log"
mkdir -p "$(dirname "$LOG_FILE")"

NAMESPACE="scality-object-storage"
SERVICE="scality-cosi-driver-metrics"
LOCAL_PORT=8080
TARGET_PORT=8080

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

echo "Fetching metrics from localhost:$LOCAL_PORT/metrics" | tee -a "$LOG_FILE"
METRICS_OUTPUT=$(curl -s http://localhost:$LOCAL_PORT/metrics | grep grpc_server_msg_sent_total)
echo "Metrics fetched successfully:" | tee -a "$LOG_FILE"
echo "$METRICS_OUTPUT" | tee -a "$LOG_FILE"

export GRPC_METRICS="$METRICS_OUTPUT"
echo "Metrics assigned to GRPC_METRICS environment variable" | tee -a "$LOG_FILE"

echo "Cleaning up port-forwarding (PID: $PORT_FORWARD_PID)" | tee -a "$LOG_FILE"
kill "$PORT_FORWARD_PID"

# Step 6: Echo the environment variable
echo "GRPC_METRICS: $GRPC_METRICS"
