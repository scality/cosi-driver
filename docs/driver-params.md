# COSI Driver Parameters

## Configuration Parameters for BucketClass

The table below details the configuration parameters for BucketClass, which determine how storage buckets and associated resources are created and managed.

| **Parameter**                                      | **Description**                                                                | **Allowed Values**         | **Required** |
|----------------------------------------------------|--------------------------------------------------------------------------------|----------------------------|--------------|
| `COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME`         | The name of the Kubernetes secret containing S3 credentials and configuration. | `string`                   | Yes          |
| `COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAMESPACE`    | The namespace in which the secret is located (e.g., `default`).                | `string` (e.g., `default`) | Yes          |

[Example](../cosi-examples/bucketclass.yaml)

## Configuration Parameters for Kubernetes secret containing S3 credentials and configuration

| **Parameter**                 | **Description**                                                                                         | **Allowed Values**                          | **Required** |
|-------------------------------|---------------------------------------------------------------------------------------------------------|---------------------------------------------|--------------|
| `COSI_S3_ACCESS_KEY_ID`       | The Access Key ID of the identity with S3 bucket creation privileges.                                   | `string`                                    | Yes          |
| `COSI_S3_SECRET_ACCESS_KEY`   | The Secret Access Key corresponding to the above Access Key ID.                                         | `string`                                    | Yes          |
| `COSI_S3_ENDPOINT`            | The S3 endpoint URL. If HTTPS is used without a TLS certificate, an insecure connection will be used.   | `string` (e.g., `https://s3.ring.internal`) | Yes          |
| `COSI_S3_REGION`              | The S3 region to use.                                                                                   | `string` (e.g., `us-east-1`)                | Yes          |
| `COSI_S3_TLS_CERT_SECRET_NAME`| The name of the secret containing the TLS certificate (optional).                                       | `string`                                    | No           |

[Example](../cosi-examples/s3-secret-for-cosi.yaml)
