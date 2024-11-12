#!/bin/bash
set -e

LOG_FILE=".github/e2e_tests/artifacts/logs/kind_cluster_logs/cosi_deployment/cleanup_debug.log"
mkdir -p "$(dirname "$LOG_FILE")"  # Ensure the log directory exists

error_handler() {
  echo "An error occurred during the COSI cleanup. Check the log file for details." | tee -a "$LOG_FILE"
  echo "Failed command: $BASH_COMMAND" | tee -a "$LOG_FILE"
  exit 1
}

trap 'error_handler' ERR

log_and_run() {
  echo "Running: $*" | tee -a "$LOG_FILE"
  "$@" | tee -a "$LOG_FILE"
}

log_and_run echo "Removing COSI driver manifests and namespace..."
log_and_run kubectl delete -k . || { echo "COSI driver manifests not found." | tee -a "$LOG_FILE"; }
log_and_run kubectl delete namespace scality-object-storage || { echo "Namespace scality-object-storage not found." | tee -a "$LOG_FILE"; }

log_and_run echo "Verifying namespace deletion..."
if kubectl get namespace scality-object-storage &>/dev/null; then
  echo "Warning: Namespace scality-object-storage was not deleted." | tee -a "$LOG_FILE"
  exit 1
fi

log_and_run echo "Removing Finalizers from Bucket Claim and Bucket"
log_and_run kubectl patch bucketclaim my-bucket-claim  -p '{"metadata":{"finalizers":[]}}' --type=merge || { echo "Bucket Claim finalizers not found." | tee -a "$LOG_FILE"; }

BUCKET_NAMES=$(kubectl get bucket -o jsonpath='{.items[*].metadata.name}')

for BUCKET_NAME in $BUCKET_NAMES; do
  log_and_run echo "Removing finalizers from bucket: $BUCKET_NAME"
  log_and_run kubectl patch bucket "$BUCKET_NAME" -p '{"metadata":{"finalizers":[]}}' --type=merge || { echo "Finalizers not found for bucket: $BUCKET_NAME" | tee -a "$LOG_FILE"; }
done

log_and_run echo "Deleting Bucket Class and Bucket Claim..."
log_and_run kubectl delete -f cosi-examples/bucketclass.yaml || { echo "Bucket Class not found." | tee -a "$LOG_FILE"; }
log_and_run kubectl delete -f cosi-examples/bucketclaim.yaml || { echo "Bucket Claim not found." | tee -a "$LOG_FILE"; }

log_and_run echo "Deleting COSI CRDs..."
log_and_run kubectl delete -k github.com/kubernetes-sigs/container-object-storage-interface-api || { echo "COSI API CRDs not found." | tee -a "$LOG_FILE"; }
log_and_run kubectl delete -k github.com/kubernetes-sigs/container-object-storage-interface-controller || { echo "COSI Controller CRDs not found." | tee -a "$LOG_FILE"; }

log_and_run echo "Verifying COSI CRDs deletion..."
if kubectl get crd | grep 'container-object-storage-interface' &>/dev/null; then
  echo "Warning: Some COSI CRDs were not deleted." | tee -a "$LOG_FILE"
  exit 1
fi

log_and_run echo "COSI cleanup completed successfully."
