#!/bin/bash

echo "Installing Ginkgo and Gomega..."
go install github.com/onsi/ginkgo/v2/ginkgo
go get github.com/onsi/gomega/...

# Create a KIND cluster
echo "Starting Minikube..."
minikube start

# echo "Logging into GHCR (GitHub Container Registry)..."
# echo "$REPOSITORY_USER_TOKEN" | docker login ghcr.io -u "$REPOSITORY_USER" --password-stdin

# Navigate to the directory and prepare the environment
echo "Preparing S3 and IAM log and directories..."
cd .github/s3_and_iam_deployment && \
mkdir -p logs/s3 logs/iam logs/cosi_driver data/vaultdb && \
sudo chown -R vscode:vscode logs data && \
chmod -R ugo+rwx logs data && \

# Pulling docker images
docker pull ghcr.io/scality/vault:7.70.26
docker pull ghcr.io/scality/cloudserver:7.70.55

# Start Docker Compose for the 'iam_s3' profile
echo "Deploying S3 and IAM using docker compose..."
docker compose --profile iam_s3 up -d

# Set Minikube's Docker environment variables
eval $(minikube docker-env)

# Prune Docker on Minikube's Docker Daemon
echo "Pruning unused images..."
docker system prune -af

echo "Setup complete."
