APP_NAME = scality-cosi-driver
BIN_DIR = ./bin

# 'go env' vars aren't always available in make environments, so get defaults for needed ones
GOARCH ?= $(shell go env GOARCH)
IMAGE_NAME ?= ghcr.io/scality/cosi:latest

.PHONY: all build test clean

all: test build

build:
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

test:
	@echo "Running Ginkgo tests..."
	# Running Ginkgo tests recursively (-r) with verbose output (-v)
	ginkgo -r -v --cover --coverprofile=coverage.txt

clean:
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)

container:
	@echo "Building container image..."
	docker build -t $(IMAGE_NAME) .
