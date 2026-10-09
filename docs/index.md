# Scality COSI Driver

The **Scality COSI Driver** integrates Scality RING Object Storage with Kubernetes, leveraging the Kubernetes Container Object Storage Interface (COSI) to enable seamless object storage provisioning and management.

## Features

| Category                  | Feature                          | Notes                                                                                                            |
|---------------------------|----------------------------------|------------------------------------------------------------------------------------------------------------------|
| **Bucket Provisioning**   | Greenfield bucket provisioning   | Creates a new S3 Bucket with default settings.                                                                  |
|                           | Brownfield bucket provisioning   | Leverages an existing bucket in S3 storage within Kubernetes workflows.                                         |
|                           | Delete Bucket                    | Deletes an S3 Bucket, but only if it is empty.                                                                  |
| **Access Management**     | Grant Bucket Access              | Provides full access to a bucket by creating new IAM credentials with access and secret keys.                   |
|                           | Revoke Bucket Access             | Removes access by deleting IAM credentials associated with the bucket.                                          |

## Quickstart Guide

To quickly deploy and test the Scality COSI Driver (for other options, see the [installation guide](installation/install-helm.md)):

1. Ensure your Kubernetes cluster is properly configured, and [Helm v3+ is installed](https://helm.sh/docs/intro/install/). The COSI specification was introduced in Kubernetes 1.25. We recommend using [one of the latest supported Kubernetes versions](https://kubernetes.io/releases/).
2. Create namespace `container-object-storage-system` and install the COSI controller deployment and COSI CRDs. The Scality COSI Driver supports only COSI **v1alpha1** — the command below pins upstream to tag `v0.2.2` (the last v1alpha1 release). Do not install from upstream `main`, which now ships v1alpha2 and is incompatible with this driver.

   ```bash
   kubectl create -k "github.com/kubernetes-sigs/container-object-storage-interface//?ref=v0.2.2"
   ```

3. Deploy the driver: Namespace `container-object-storage-system` will be created in step 2.

   ```bash
   helm install scality-cosi-driver oci://ghcr.io/scality/cosi-driver/helm-charts/scality-cosi-driver \
       --namespace container-object-storage-system
   ```

4. Verify the deployment:

   ```bash
   kubectl get pods -n container-object-storage-system
   ```

   There should be 2 pods `Running` in the `container-object-storage-system` namespace:

   ```sh
   $ kubectl get pods -n container-object-storage-system
   NAME                                                   READY   STATUS    RESTARTS   AGE
   container-object-storage-controller-7f9f89fd45-h7jtn   1/1     Running   0          25h
   scality-cosi-driver-67d96bf8ff-9f59l                   2/2     Running   0          20h
   ```

To learn how to use the COSI driver, refer to the [Usage documentation](Usage.md).

## Compatibility

| S3 Implementation | Supported Versions | Notes                            |
|-------------------|--------------------|----------------------------------|
| Scality RING      | 9.4.2 or newer     | Full support with driver v1.x    |

## Support

For issues, please create a ticket in the [GitHub Issues](https://github.com/scality/cosi-driver/issues) section.
