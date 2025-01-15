# Scality COSI Driver Usage Guide: Bucket Provisioning & Access Control

This document provides an overview and step-by-step guidance for implementing **Bucket Provisioning** (both Greenfield and Brownfield) and **Access Control** using the Scality COSI Driver. Example YAML manifests can be found in the [cosi-examples](../cosi-examples/) folder.

> **Note**
> The Scality COSI Driver supports standard AWS S3 and IAM compliant storage solutions like Scality RING, Scality ARTESCA and AWS S3 & IAM

## Prerequisites

Before proceeding, ensure that the following components are installed on your cluster:

1. **Kubernetes Container Object Storage Interface (COSI) CRDs**
2. **Container Object Storage Interface Controller**

Refer to the quick start guide in the [README](../README.md#quickstart-guide) for installation instructions.

### Common Setup Steps

1. **Create the IAM User (by the Storage Administrator)**

   Create an IAM user and a paid of Access Key ID and Secret Access Key. This user will be used by COSI driver. Assign S3/IAM permissions that allow bucket creation and user management. Permissions needed by COSI driver:
     - `S3:CreateBucket`
     - `S3:DeleteBucket`
     - `IAM:GetUser`
     - `IAM:CreateUser`
     - `IAM:DeleteUser`
     - `IAM:PutUserPolicy`
     - `IAM:DeleteUserPolicy`
     - `IAM:ListAccessKeys`
     - `IAM:CreateAccessKey`
     - `IAM:DeleteAccessKey`

2. **Collect Access & Endpoint Details**

   The Storage Administrator provides the below details to the Kubernetes Administrator:

   - S3 endpoint (and IAM endpoint, if different)
   - Region
   - Access Key ID & Secret Key
   - `tlsCert`, if using HTTPS

3. **Create a Kubernetes Secret (by the Kubernetes Administrator)**

   The Kubernetes Administrator creates a secret containing the above credentials and configuration details:

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: v1
   kind: Secret
   metadata:
     name: s3-secret-for-cosi
   type: Opaque
   stringData:
     accessKeyId: <ACCESS_KEY_ID>
     secretAccessKey: <SECRET_ACCESS_KEY>
     endpoint: <S3_ENDPOINT>
     region: <REGION>
     iamEndpoint: <OPTIONAL_IAM_ENDPOINT>
     tlsCert: |-
       -----BEGIN CERTIFICATE-----
       ...
       -----END CERTIFICATE-----
   EOF
   ```

   > **Note**
   > Update `<ACCESS_KEY_ID>`, `<SECRET_ACCESS_KEY>`, `<S3_ENDPOINT>`, `<REGION>`, with valid values for your environment. If your endpoint does not require a TLS cert, you can remove it. Similarly, add IAM endpoint with `iamEndpoint` if its different from S3 endpoint otherwise remove it.
   > If using TLS cert, include the certificate content (PEM-encoded) in the stringData section of the Secret. Use a multi-line block scalar (|-) in YAML so that the certificate (with newlines) is preserved correctly.

---

## 1. Bucket Provisioning

In the **Scality COSI Driver**, both **Greenfield** and **Brownfield** provisioning share similar steps, with minor differences in how resources (Bucket, BucketClaim) are created.

> Note:
> For **fully working** examples, see the YAMLs in the [cosi-examples/brownfield](./cosi-examples/brownfield/) and [cosi-examples/greenfield](./cosi-examples/greenfield/) directories.
> For brownfield scenario it is madatory to create COSI CRs in the same namespace as COSI driver and controller.

### 1.1 Greenfield: Creating a New Bucket

Greenfield provisioning will create a brand-new S3 bucket in your object store, managed by Kubernetes. Examples can be found [here](../cosi-examples/greenfield/).

1. **Create a BucketClass**
   A `BucketClass` defines how buckets should be provisioned or deleted. The bucket class name is used as a prefix for bucket name by COSI:

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: objectstorage.k8s.io/v1alpha1
   kind: BucketClass
   metadata:
     name: greenfield-bucketclass
     namespace: container-object-storage-system
   driverName: cosi.scality.com
   deletionPolicy: Delete
   parameters:
     objectStorageSecretName: s3-secret-for-cosi
     objectStorageSecretNamespace: default
   EOF
   ```

   - `driverName` must match the Scality driver (default: `cosi.scality.com`). This determined by driver-prefix in the COSI driver deployment. For more information check [COSI Driver Parameters](./docs/driver-params.md).
   - `deletionPolicy` can be `Delete` or `Retain`.
   - `objectStorageSecretName` and `objectStorageSecretNamespace` reference the secret you created earlier in [Common Setup Steps](#common-setup-steps). For more information check [COSI Driver Parameters](./docs/driver-params.md).

2. **Create a BucketClaim**

   A `BucketClaim` requests a new bucket using the above `BucketClass`:

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: objectstorage.k8s.io/v1alpha1
   kind: BucketClaim
   metadata:
     name: my-greenfield-bucketclaim
     namespace: container-object-storage-system
   spec:
     bucketClassName: greenfield-bucketclass
     protocols:
       - S3
   EOF
   ```

   - This will automatically provision a new S3 bucket in the backend using the COSI driver. The COSI driver will use credentials and endpoint mentioned in the secret specified in the `BucketClass` to create the bucket.
   - The actual bucket name on S3 is typically generated by the driver (e.g., `<bucketclassName>-<UUID>`).
   - Only `S3` protocol is supported at the moment.

### 1.2 Brownfield: Using an Existing Bucket

Brownfield provisioning allows you to manage an **already-existing** S3 bucket in Kubernetes.

> Note: For brownfield scenario, COSI CRs for Bucket and Access provisioning should be created in the same namespace as COSI driver and controller.

1. **Verify Existing Bucket**

   Ensure the bucket already exists in S3 either through Storage Administrator or by running the following AWS CLI command:

   ```bash
   aws s3api head-bucket --bucket <EXISTING_BUCKET_NAME> --endpoint-url <S3_ENDPOINT>
   ```

2. **Create a BucketClass**

   Similar to Greenfield, but you will typically still reference the same secret:

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: objectstorage.k8s.io/v1alpha1
   kind: BucketClass
   metadata:
     name: brownfield-bucketclass
     namespace: container-object-storage-system
   driverName: cosi.scality.com
   deletionPolicy: Delete
   parameters:
     objectStorageSecretName: s3-secret-for-cosi
     objectStorageSecretNamespace: default
   EOF
   ```

   > **Note**
   > For Brownfield, existing buckets when imported using the steps below, do not follow the `deletionPolicy` even if it set to `Delete` . All buckets created using this bucket class for greenfield scenario will still respect the `deletionPolicy`

3. **Create the Bucket Instance**
   This is where we tell Kubernetes about the existing bucket:

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: objectstorage.k8s.io/v1alpha1
   kind: Bucket
   metadata:
     name: "<EXISTING_BUCKET_NAME>"
     namespace: container-object-storage-system
   spec:
     bucketClaim: {}
     driverName: cosi.scality.com
     bucketClassName: brownfield-bucketclass
     driverName: cosi.scality.com
     deletionPolicy: Retain
     existingBucketID: "<EXISTING_BUCKET_NAME>"
     parameters:
       objectStorageSecretName: s3-secret-for-cosi
       objectStorageSecretNamespace: default
     protocols:
       - S3
   EOF
   ```

   - `name` and `existingBucketID` should be the same as the existing bucket name in S3 storage.

4. **Create the BucketClaim**
   Reference the existing `Bucket` object by name via `existingBucketName`:

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: objectstorage.k8s.io/v1alpha1
   kind: BucketClaim
   metadata:
     name: my-brownfield-bucketclaim
     namespace: container-object-storage-system
   spec:
     bucketClassName: brownfield-bucket-class
     existingBucketName: "<EXISTING_BUCKET_NAME>"
     protocols:
       - S3
   EOF
   ```

    - `existingBucketName` should match the `name` of the `Bucket` object created in the previous step for Bucket Instance.

### Bucket Provisioning Cleanup

To remove the buckets and associated objects:

- **Greenfield**:

  ```bash
  kubectl delete bucketclaim my-greenfield-bucketclaim
  ```

  - Deleting the `BucketClaim` will remove the underlying bucket only if:
    - `deletionPolicy` was set to `Delete` in `BucketClass`.
    - The bucket is empty at the time of deletion.

- **Brownfield**:

  ```bash
  kubectl delete bucketclaim my-brownfield-bucketclaim
  ```

  - Deleting the `BucketClaim` and `Bucket` objects in Kubernetes **does not** delete the actual pre-existing bucket in S3 even if if `deletionPolicy` is `Delete`.

---

## 2. Access Control (Common to Greenfield & Brownfield)

Access Control configuration is effectively the same for both Greenfield and Brownfield. Once the `BucketClaim` is ready, you can request credentials for the bucket via a `BucketAccess` resource.

### 2.1 Create a BucketAccessClass

A `BucketAccessClass` defines how access (IAM policy or S3 keys) is granted:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketAccessClass
metadata:
  name: bucketaccessclass
  namespace: container-object-storage-system
spec:
  driverName: cosi.scality.com
  authenticationType: KEY
  parameters:
    objectStorageSecretName: s3-secret-for-cosi
    objectStorageSecretNamespace: default
EOF
```

> **Note**
>
> - `authenticationType` is often `Key` for basic S3 key-based credentials.
> - `objectStorageSecretName` and `objectStorageSecretNamespace` reference the secret you created earlier in [Common Setup Steps](#common-setup-steps).

### 2.2 Request Bucket Access

Once the `BucketClaim` is bound (Greenfield or Brownfield), create a `BucketAccess` to generate a credential secret in the cluster:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketAccess
metadata:
  name: my-bucketaccess
  namespace: container-object-storage-system
spec:
  bucketClaimName: my-greenfield-bucketclaim  # or my-brownfield-bucketclaim
  bucketAccessClassName: bucketaccessclass
  credentialsSecretName: my-s3-credentials
  protocol: S3
EOF
```

- `bucketClaimName` references the claim from the previous section.
- `credentialsSecretName` is the name of the secret that will contain the newly generated credentials (Access Key ID / Secret Key).

Once the `BucketAccess` is created, the driver will:

- Create IAM user.
- Generate an inline policy with name from bucket name, for the user with the correct full S3 on the associated bucket.
- Create a new Kubernetes `Secret` (`my-s3-credentials`) with the S3 Access Key ID and Secret Key for clients. This will follow the [BucketInfo format](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/1979-object-storage-support/README.md#bucketinfo).

### 2.3 Revoking Access

To revoke access for a user or application:

```bash
kubectl delete bucketaccess my-bucketaccess
```

This triggers the removal of the IAM user/access keys on the S3 side.

---

## Validation & Troubleshooting

- **Bucket Verification**:
  Confirm the bucket exists (or is deleted) in S3 as expected.
- **IAM User Verification**:
  Check that the necessary IAM user/policies are created or removed upon `BucketAccess` creation/deletion.
- **Kubernetes Secret**:
  Inspect the auto-generated `credentialsSecretName` to see if keys have been populated:

  ```bash
  kubectl get secret my-s3-credentials -o yaml
  ```

---

## Further Reading

- [Official COSI Documentation](https://github.com/kubernetes-sigs/container-object-storage-interface-api)
- [Scality COSI Driver Parameters](./docs/driver-params.md)
- [cosi-examples Folder](./cosi-examples/) for fully working YAML samples.
