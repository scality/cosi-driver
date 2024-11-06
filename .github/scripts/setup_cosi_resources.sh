#!/bin/bash
set -e  # Exit on any command failure

# Define log file for debugging
LOG_FILE=".github/e2e_tests/artifacts/logs/kind_cluster_logs/cosi_deployment/setup_debug.log"
mkdir -p "$(dirname "$LOG_FILE")"  # Ensure the log directory exists

# Error handling function
error_handler() {
  echo "An error occurred during the COSI setup. Check the log file for details." | tee -a "$LOG_FILE"
  echo "Failed command: $BASH_COMMAND" | tee -a "$LOG_FILE"
  exit 1
}

# Trap errors and call the error handler
trap 'error_handler' ERR

# Log command execution to the log file for debugging
log_and_run() {
  echo "Running: $*" | tee -a "$LOG_FILE"
  if ! "$@" | tee -a "$LOG_FILE"; then
    echo "Error: Command failed - $*" | tee -a "$LOG_FILE"
    exit 1
  fi
}

# Step 1: Install COSI CRDs
log_and_run echo "Installing COSI CRDs..."
log_and_run kubectl create -k github.com/kubernetes-sigs/container-object-storage-interface-api
log_and_run kubectl create -k github.com/kubernetes-sigs/container-object-storage-interface-controller

# Step 2: Verify COSI Controller Pod Status
log_and_run echo "Verifying COSI Controller Pod status..."
if ! kubectl wait --namespace default --for=condition=ready pod -l app.kubernetes.io/name=container-object-storage-interface-controller --timeout=10s; then
  echo "Error: COSI Controller pod did not reach ready state." | tee -a "$LOG_FILE"
  exit 1
fi
log_and_run kubectl get pods --namespace default

# Step 3: Build COSI driver Docker image
log_and_run echo "Building COSI driver image..."
log_and_run docker build -t ghcr.io/scality/cosi:latest .

# Step 4: Load COSI driver image into KIND cluster
log_and_run echo "Loading COSI driver image into KIND cluster..."
log_and_run kind load docker-image ghcr.io/scality/cosi:latest --name object-storage-cluster

# Step 5: Run COSI driver
log_and_run echo "Applying COSI driver manifests..."
if ! kubectl apply -k .; then
  echo "Error: Failed to apply COSI driver manifests." | tee -a "$LOG_FILE"
  exit 1
fi

# Step 6: Verify COSI driver Pod Status
log_and_run echo "Verifying COSI driver Pod status..."
if ! kubectl wait --namespace scality-object-storage --for=condition=ready pod --selector=app.kubernetes.io/name=scality-cosi-driver --timeout=20s; then
  echo "Error: COSI driver Pod did not reach ready state." | tee -a "$LOG_FILE"
  exit 1
fi
log_and_run kubectl get pods -n scality-object-storage

log_and_run echo "COSI setup completed successfully."
