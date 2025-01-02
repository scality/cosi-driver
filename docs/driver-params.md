# COSI Driver Parameters

## Configuration Parameters for BucketClass

The table below details the configuration parameters for BucketClass, which determine how storage buckets and associated resources are created and managed.

| **Parameter**                                      | **Description**                                                                | **Allowed Values**         | **Required** |
|----------------------------------------------------|--------------------------------------------------------------------------------|----------------------------|--------------|
| `objectStorageSecretName`         | The name of the Kubernetes secret containing S3 credentials and configuration. | `string`                   | Yes          |
| `objectStorageSecretNamespace`    | The namespace in which the secret is located (e.g., `default`).                | `string` (e.g., `default`) | Yes          |

[Example](../cosi-examples/greenfield/bucketclass.yaml)

## Configuration Parameters for Kubernetes secret containing S3 credentials and configuration

| **Parameter**                 | **Description**                                                                                         | **Allowed Values**                          | **Required** |
|-------------------------------|---------------------------------------------------------------------------------------------------------|---------------------------------------------|--------------|
| `accessKeyId`       | The Access Key ID of the identity with S3 bucket creation privileges.                                   | `string`                                    | Yes          |
| `secretAccessKey`   | The Secret Access Key corresponding to the above Access Key ID.                                         | `string`                                    | Yes          |
| `endpoint`            | The S3 endpoint URL. If HTTPS is used without a TLS certificate, an insecure connection will be used.   | `string` (e.g., `https://s3.ring.internal`) | Yes          |
| `region`              | The S3 region to use.                                                                                   | `string` (e.g., `us-east-1`)                | Yes          |
| `tlsCert`| The name of the secret containing the TLS certificate (optional).                                       | `string`                                    | No           |

[Example](../cosi-examples/s3-secret-for-cosi.yaml)

## Deployment Parameters for COSI Driver

Below are the deployment parameters for configuring the COSI driver, which can be passed as flags or environment variables.

| **Parameter**                   | **Description**                                                | **Default Value**                | **Required** |
|---------------------------------|----------------------------------------------------------------|----------------------------------|--------------|
| `driver-address`                | The socket file address for the COSI driver.                   | `unix:///var/lib/cosi/cosi.sock` | Yes          |
| `driver-prefix`                 | The prefix for the COSI driver (e.g., `<prefix>.scality.com`). | `cosi`                           | No           |
| `driver-metrics-address`        | The address to expose Prometheus metrics.                      | `:8080`                          | No           |
| `driver-metrics-path`           | The HTTP path for exposing metrics.                            | `/metrics`                       | No           |
| `driver-custom-metrics-prefix`  | The prefix for metrics collected by the COSI driver.           | `scality_cosi_driver`            | No           |

### Notes

- If driver-metrics-path does not start with /, it will automatically prepend /.
- Prometheus metrics are exposed for monitoring at the address and path specified.
