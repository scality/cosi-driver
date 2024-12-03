# COSI Driver Parameters

## Configuration Parameters for BucketClass

The table below details the configuration parameters for BucketClass, which determine how storage buckets and associated resources are created and managed.

| **Parameter**                                      | **Description**                                                                | **Allowed Values**         | **Required** |
|----------------------------------------------------|--------------------------------------------------------------------------------|----------------------------|--------------|
| `objectStorageSecretName`         | The name of the Kubernetes secret containing S3 credentials and configuration. | `string`                   | Yes          |
| `objectStorageSecretNameSpace`    | The namespace in which the secret is located (e.g., `default`).                | `string` (e.g., `default`) | Yes          |

[Example](../cosi-examples/bucketclass.yaml)

## Configuration Parameters for Kubernetes secret containing S3 credentials and configuration

| **Parameter**                 | **Description**                                                                                         | **Allowed Values**                          | **Required** |
|-------------------------------|---------------------------------------------------------------------------------------------------------|---------------------------------------------|--------------|
| `accessKeyId`       | The Access Key ID of the identity with S3 bucket creation privileges.                                   | `string`                                    | Yes          |
| `secretAccessKey`   | The Secret Access Key corresponding to the above Access Key ID.                                         | `string`                                    | Yes          |
| `endpoint`            | The S3 endpoint URL. If HTTPS is used without a TLS certificate, an insecure connection will be used.   | `string` (e.g., `https://s3.ring.internal`) | Yes          |
| `region`              | The S3 region to use.                                                                                   | `string` (e.g., `us-east-1`)                | Yes          |
| `tlsCert`| The name of the secret containing the TLS certificate (optional).                                       | `string`                                    | No           |

[Example](../cosi-examples/s3-secret-for-cosi.yaml)
