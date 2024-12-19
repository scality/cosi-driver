#!/bin/bash
set -e

LOG_FILE=".github/e2e_tests/artifacts/logs/e2e_tests/metrics_service.log"
mkdir -p "$(dirname "$LOG_FILE")"

NAMESPACE="scality-object-storage"
SERVICE="scality-cosi-metrics"
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

# Log command execution to the log file for debugging
log_and_run() {
  "$@" 2>&1 | tee -a "$LOG_FILE"
}

log_and_run kubectl describe svc scality-cosi-driver-metrics -n scality-object-storage

log_and_run kubectl port-forward svc/scality-cosi-driver-metrics -n scality-object-storage 8080:8080 &
PORT_FORWARD_PID=$!

if ps -p $PORT_FORWARD_PID > /dev/null; then
    log_and_run echo "Port-forwarding established. Querying metrics..."

    log_and_run curl -f http://localhost:$LOCAL_PORT/metrics > metrics_output.txt

    if [ $? -eq 0 ]; then
        log_and_run  echo "Metrics fetched successfully. Metrics output:"
        log_and_run cat metrics_output.txt
        METRICS_WORKING=true
    else
        log_and_run echo "Failed to fetch metrics from http://localhost:$LOCAL_PORT/metrics"
        METRICS_WORKING=false
    fi

    log_and_run kill $PORT_FORWARD_PID
else
    log_and_run echo "Port-forwarding failed to establish."
    METRICS_WORKING=false
fi

if [ "$METRICS_WORKING" = true ]; then
    log_and_run echo "Metrics service is working as expected."
    exit 0
else
    log_and_run echo "Metrics service is not working as expected."
    exit 1
fi

