#!/bin/bash
set -e

# Create a directory to store the logs
mkdir -p logs/kind_cluster_logs
LOG_FILE_PATH=".github/e2e_tests/artifacts/logs/kind_cluster_logs"
mkdir -p "$(dirname "$LOG_FILE_PATH")"  # Ensure the log directory exists
# Define namespaces to capture logs from
namespaces=("default" "scality-object-storage")

# Loop through specified namespaces, pods, and containers
for namespace in "${namespaces[@]}"; do
  for pod in $(kubectl get pods -n $namespace -o jsonpath='{.items[*].metadata.name}'); do
    for container in $(kubectl get pod $pod -n $namespace -o jsonpath='{.spec.containers[*].name}'); do
      # Capture current logs for each container
      kubectl logs -n $namespace $pod -c $container > ${LOG_FILE_PATH}/_${namespace}_${pod}_${container}.log 2>&1
      # Capture previous logs if the container has restarted
      kubectl logs -n $namespace $pod -c $container --previous > ${LOG_FILE_PATH}/${namespace}_${pod}_${container}_previous.log 2>&1 || true
    done
  done
done
