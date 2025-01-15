# Scality COSI driver

The Scality COSI Driver integrates Scality RING Object Storage (and AWS S3-and-IAM compatible storage solutions) with Kubernetes, leveraging the Kubernetes Container Object Storage Interface (COSI) to enable seamless object storage provisioning and management. This repository provides all necessary resources to deploy, use, and contribute to the Scality COSI Driver.

## Overview

The Scality COSI Driver allows Kubernetes users to:

- **Dynamically Provision Buckets**: Create and delete object storage buckets on-demand.
- **Manage Access Policies**: Control bucket access through Kubernetes CRDs.
- **Integrate with S3 and IAM compatible Object Storage**: Leverage Scality's robust and scalable storage backends.
- **Utilize Standard Kubernetes Tools**: Manage object storage using familiar Kubernetes APIs and tooling.






## Documentation

Comprehensive documentation is available in the [`docs/`](docs/README.md) directory. It includes detailed guides and references.


### Installation Guides

- [Install with Helm from OCI](docs/installation/install-helm-oci.md)
- [Install with Helm Locally](docs/installation/install-helm-local.md)
- [Install with Kustomize on Minikube](docs/installation/install-kustomize.md)
- [Setup with Development Containers](docs/installation/install-dev-containers.md)

### Testing Procedures

- [Run End-to-End Tests Locally](docs/testing/test-e2e-local.md)
- [Testing on Minikube with Helm](docs/testing/test-minikube-helm.md)
- [Testing on Minikube with Kustomize](docs/testing/test-minikube-kustomize.md)

### Configuration Guides

- [Customization Guide](docs/configuration/config-customization.md)
- [Values.yaml Reference](docs/configuration/config-values-reference.md)

### Development Resources

- [Development Container Setup](docs/development/dev-container-setup.md)
- [Developer Guide](docs/development/dev-guide.md)

## Quick Start

To quickly install the Scality COSI Driver using Helm from an OCI registry, follow these steps:

### Prerequisites

- **Kubernetes Cluster**: Running version 1.23 or later.
- **Helm**: Version 3.8.0 or later.

### Installation Steps

1. **Ensure Helm is Installed**

   Verify your Helm installation:

   ```bash
   helm version --short
   # Expected output: v3.8.0+ or later
   ```

2. **Install the COSI Driver**

   Install the driver using the OCI registry:

   ```bash
   helm install scality-cosi-driver oci://registry.scality.com/charts/cosi-driver
   ```

3. **Verify the Installation**

   Check that the driver pods are running:

   ```bash
   kubectl get pods -n scality-cosi
   ```

   Confirm the COSI driver is registered:

   ```bash
   kubectl get csidrivers
   ```

For more detailed instructions and alternative installation methods, please refer to the [Installation Guides](docs/installation/).

