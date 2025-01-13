#!/bin/bash
set -e

NAMESPACE="container-object-storage-system"

echo "Verifying Helm installation..."

if ! helm status scality-cosi-driver -n $NAMESPACE; then
  echo "Helm release scality-cosi-driver not found in namespace $NAMESPACE"
  exit 1
fi

echo "Verifying COSI driver Pod status for 120s..."
if ! kubectl wait --namespace $NAMESPACE --for=condition=ready pod --selector=app.kubernetes.io/name=scality-cosi-driver --timeout=120s; then
  echo "Error: COSI driver Pod did not reach ready state."
  kubectl get pods -n $NAMESPACE
  kubectl describe --namespace $NAMESPACE pod --selector=app.kubernetes.io/name=scality-cosi-driver
  exit 1
fi
kubectl get pods -n $NAMESPACE

echo "Helm installation verified successfully."
