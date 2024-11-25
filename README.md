Certainly! Below is the directory structure including the root of your repository and the contents of the root `README.md`. The other documentation files are organized under the `docs/` directory in subfolders like `installation`, `configuration`, and `development`.

---

### **Directory Structure**

```
├── docs/
│   ├── README.md                     # Overview of all documentation
│   ├── installation/
│   │   ├── install-helm-oci.md       # Installing with Helm from OCI
│   │   ├── install-helm-local.md     # Installing with Helm locally
│   │   ├── install-kustomize.md      # Installing with Kustomize on Minikube
│   │   └── install-dev-containers.md # Using development containers for setup
│   ├── testing/
│   │   ├── test-e2e-local.md         # Running end-to-end tests locally
│   │   ├── test-minikube-helm.md     # Testing on Minikube with Helm
│   │   └── test-minikube-kustomize.md# Testing on Minikube with Kustomize
│   ├── configuration/
│   │   ├── config-customization.md   # Guide to customizing configuration values
│   │   └── config-values-reference.md# Detailed reference for `values.yaml`
│   └── development/
│       ├── dev-container-setup.md    # Setting up development containers
│       └── dev-guide.md              # Developer guide for contributing
├── charts/                           # Helm charts for deployment
│   └── cosi-driver/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
├── src/                              # Source code files
│   └── ...
├── tests/                            # Test files and scripts
│   └── ...
└── ...                               # Other files and directories
```

---

### **Contents of the Root `README.md`**

```markdown
# Scality COSI Driver

Welcome to the **Scality COSI Driver** repository! This project provides a Container Object Storage Interface (COSI) driver implementation for Scality's storage solutions, enabling seamless management of object storage resources within Kubernetes clusters.

## Overview

The Scality COSI Driver allows Kubernetes users to:

- **Dynamically Provision Buckets**: Create and delete object storage buckets on-demand.
- **Manage Access Policies**: Control bucket access through Kubernetes CRDs.
- **Integrate with Scality Storage**: Leverage Scality's robust and scalable storage backends.
- **Utilize Standard Kubernetes Tools**: Manage object storage using familiar Kubernetes APIs and tooling.

## Table of Contents

- [Documentation](#documentation)
- [Quick Start](#quick-start)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)
- [Acknowledgments](#acknowledgments)

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/README.md) directory. It includes detailed guides and references:

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
   helm install scality-cosi-driver oci://registry.scality.com/charts/cosi-driver --version 0.1.0-beta
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
