# Installing the Scality COSI Driver with Helm

This guide provides step-by-step instructions for installing the Scality COSI Driver using Helm. You can choose to install the chart either locally from your machine or directly from an OCI registry. This unified guide covers both methods.

---

## Table of Contents

- [Installing the Scality COSI Driver with Helm](#installing-the-scality-cosi-driver-with-helm)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Installation Methods](#installation-methods)
    - [Install locally without helm package](#install-locally-without-helm-package)
    - [Package locally and install](#package-locally-and-install)
    - [Install from OCI Registry with Helm](#install-from-oci-registry-with-helm)
  - [Verifying the Installation](#verifying-the-installation)
  - [Uninstalling the Chart](#uninstalling-the-chart)
  - [Troubleshooting](#troubleshooting)
  - [Additional Resources](#additional-resources)

---

## Prerequisites

- **Kubernetes Cluster**: Ensure you have access to a running Kubernetes cluster (v1.23 or later).
- **Helm**: Install Helm v3.8.0 or later.
- **Git** (for local installation): Installed on your local machine to clone the repository.
- **OCI Registry Access** (for OCI installation): Access to the OCI registry where the Helm chart is hosted (e.g., GitHub Container Registry).

---

## Installation Methods

### Install locally without helm package

```bash
    git clone https://github.com/scality/cosi-driver.git
    cd cosi-driver
    helm install scality-cosi-driver ./helm/scality-cosi-driver --namespace scality-object-storage --create-namespace --set image.tag=0.1.0
```

### Package locally and install

```bash
    git clone https://github.com/scality/cosi-driver.git
    cd cosi-driver
    helm package ./helm/scality-cosi-driver --version 0.1.0
    helm install scality-cosi-driver ./scality-cosi-driver-0.1.0.tgz --namespace scality-object-storage --create-namespace --set image.tag=0.1.0
```

### Install from OCI Registry with Helm

```bash
    helm install scality-cosi-driver oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --version 0.0.1 --namespace scality-cosi --create-namespace --set image.tag=0.1.0
```

---

## Verifying the Installation

After installing the chart using either method, verify that the Scality COSI Driver is running correctly.

1. **Check the Pods in the Namespace**

   ```bash
   kubectl get pods -n scality-cosi
   ```

2. **Check the COSI Driver Registration**

   ```bash
   kubectl get csidrivers
   ```

   You should see an entry for `scality-cosi-driver`.

3. **Describe the Deployment**

   ```bash
   kubectl describe deployment scality-cosi-driver -n scality-cosi
   ```

4. **Check Logs**

   If there are issues, check the logs of the driver pod:

   ```bash
   kubectl logs -l app.kubernetes.io/name=scality-cosi-driver -n scality-cosi
   ```

---

## Uninstalling the Chart

To uninstall the Scality COSI Driver and remove all associated resources:

```bash
helm uninstall scality-cosi-driver --namespace scality-cosi
```

Optionally, delete the namespace if it's no longer needed:

```bash
kubectl delete namespace scality-cosi
```

---

## Troubleshooting

- **Helm Version Issues**: Ensure you're using Helm v3.8.0 or later for OCI support.
- **OCI Authentication Errors**: Verify your credentials and ensure they have the necessary permissions.
- **Network Issues**: Ensure your network allows access to the OCI registry.
- **Resource Conflicts**: Check for existing resources that might conflict with the installation.
- **Logs**: Always check the pod logs for error messages if the driver is not running as expected.
- **Log in to the OCI Registry**: Log in to the `ghcr.io` using Helm: `helm registry login -u <username> -p <password> ghcr.io`
- **Chart debuggeing**: View chart details using `helm show all oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --version <chart-version>`
- Templating the chart**: To render the Helm templates and see the Kubernetes resources that will be created: `helm template scality-cosi-driver oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --version <chart-version>`

---

## Additional Resources

- **Scality COSI Driver GitHub Repository**: [https://github.com/scality/cosi-driver](https://github.com/scality/cosi-driver)
- **Helm Documentation**: [https://helm.sh/docs/](https://helm.sh/docs/)
- **OCI Support in Helm**: [Helm OCI Documentation](https://helm.sh/docs/topics/registries/)
- **Kubernetes Documentation**: [https://kubernetes.io/docs/home/](https://kubernetes.io/docs/home/)

---

When a new release of the Scality COSI Driver is published, it includes:

- A Docker image pushed to `ghcr.io/scality/cosi-driver:<tag>`
- A Helm chart available in the OCI registry `ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver`

**Note**: Always replace placeholders like `<username>`, `<password>`, and `<chart-version>` with your actual credentials and desired versions.
