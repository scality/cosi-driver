# Installing the Scality COSI Driver with Helm

This guide provides step-by-step instructions for installing the Scality COSI Driver using Helm. You can choose to install the chart either locally from your machine or directly from an OCI registry. This unified guide covers both methods.

---

## Prerequisites

- **Kubernetes Cluster**: Ensure you have access to a running Kubernetes cluster (v1.23 or later).
- **Helm**: Install Helm v3.8.0 or later.
- **Git** (for local installation): Installed on your local machine to clone the repository.
- **OCI Registry Access** (for OCI installation): Access to the OCI registry where the Helm chart is hosted (e.g., GitHub Container Registry).

---

## Installation Methods

You can install the Scality COSI Driver using Helm in multiple ways. Choose the method that best suits your environment and requirements.
Its recommended to deploy COSI controller first which creates the `container-object-storage-system` namespace and then install the COSI driver. If the namespace is not created, the COSI driver installation will fail. Use `--create-namespace` flag to create the namespace if it does not exist.

### Deploy COSI controller and related CRDs

```bash
kubectl create -k github.com/kubernetes-sigs/container-object-storage-interface
```


### Install from OCI Registry with Helm

To automatically install the latest release of cosi-driver:

```bash
    helm install scality-cosi-driver oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --namespace container-object-storage-system --create-namespace
```

Or find a desired version listed in the [releases] and
specify it when installing:

```bash
    helm install scality-cosi-driver oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --namespace container-object-storage-system --create-namespace --version <tag>
```

### Install locally without helm package

Find the latest or a desired version available under [releases].

```bash
    git clone https://github.com/scality/cosi-driver.git
    cd cosi-driver
    # replace <tag> with the latest version
    helm install scality-cosi-driver ./helm/scality-cosi-driver --namespace container-object-storage-system --create-namespace --set image.tag=<tag>
```

### Package locally and install

Find the latest or a desired version available under [releases].

```bash
    git clone https://github.com/scality/cosi-driver.git
    cd cosi-driver
    # replace <tag> with the latest version
    helm package ./helm/scality-cosi-driver --version <tag> --app-version <tag>
    helm install scality-cosi-driver ./scality-cosi-driver-<tag>.tgz --namespace container-object-storage-system --create-namespace
```

---

## Verifying the Installation

After installing the chart using either method, verify that the Scality COSI Driver is running correctly.

1. **Check the Pods in the Namespace**

   ```bash
   kubectl get pods -n container-object-storage-system
   ```

   You should see a pod for `scality-cosi-driver`.

2. **Check Logs and Deployment events**

   If there are issues, check if the events of the deployment for any errors

   ```bash
   kubectl describe deployment scality-cosi-driver --namespace container-object-storage-system
   ```

   If the Pod is running check the logs for any errors

   ```bash
   kubectl logs -l app.kubernetes.io/name=scality-cosi-driver --namespace container-object-storage-system
   ```

---

## Uninstalling the Chart

To uninstall the Scality COSI Driver and remove all associated resources:

```bash
helm uninstall scality-cosi-driver --namespace container-object-storage-system
```

---

## Troubleshooting

- **Helm Version Issues**: Ensure you're using Helm v3.8.0 or later for OCI support.
- **OCI Authentication Errors**: Verify your credentials and ensure they have the necessary permissions.
- **Network Issues**: Ensure your network allows access to the OCI registry.
- **Resource Conflicts**: Check for existing resources that might conflict with the installation.
- **Logs**: Always check the pod logs for error messages if the driver is not running as expected.
- **Chart debugging**: View chart details using `helm show all oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --version <chart-version>`
- **Templating the chart**: To render the Helm templates and see the Kubernetes resources that will be created: `helm template scality-cosi-driver oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver --version <chart-version>`

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

[releases]: https://github.com/scality/cosi-driver/releases
