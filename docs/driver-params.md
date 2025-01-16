# COSI Driver Parameters

## Configuration Parameters for BucketClass

The table below details the configuration parameters for BucketClass, which determine how storage buckets and associated resources are created and managed.

| **Parameter**                                      | **Description**                                                                | **Allowed Values**         | **Required** |
|----------------------------------------------------|--------------------------------------------------------------------------------|----------------------------|--------------|
| `objectStorageSecretName`         | The name of the Kubernetes secret containing S3 credentials and configuration. | `string`                   | Yes          |
| `objectStorageSecretNamespace`    | The namespace in which the secret is located (e.g., `default`).                | `string` (e.g., `default`) | Yes          |

[Example](../cosi-examples/greenfield/bucketclass.yaml)

## Configuration Parameters for Kubernetes secret containing S3 credentials and configuration

| **Parameter**                 | **Description**                                                                                 | **Allowed Values**                           | **Required** |
|-------------------------------|---------------------------------------------------------------------------------------------------------|---------------------------------------------|--------------|
| `accessKeyId`       | The Access Key ID of the identity with S3 bucket creation privileges.                                     | `string`                                     | Yes          |
| `secretAccessKey`   | The Secret Access Key corresponding to the above Access Key ID.                                           | `string`                                     | Yes          |
| `endpoint`            | The S3 endpoint URL. If HTTPS is used without a TLS certificate, an insecure connection will be used.   | `string` (e.g., `https://s3.ring.internal`)  | Yes          |
| `region`              | The S3 region to use.                                                                                   | `string` (e.g., `us-east-1`)                 | Yes          |
| `tlsCert`| PEM encoded TLS certificate (optional).                                                                              | `string`                                     | No           |
| `iamEndpoint`        | The IAM endpoint URL. If not specified endpoint is used as IAMendpoint                                   | `string` (e.g., `https://iam.ring.internal`) | No           |

[Example](../cosi-examples/s3-secret-for-cosi.yaml)

## Deployment Parameters for the Scality COSI Driver

Below are the deployment parameters for configuring the COSI driver, which can be passed as flags or environment variables.

| **Parameter**                   | **Description**                                                                               | **Default Value**                    | **Required** |
|---------------------------------|-----------------------------------------------------------------------------------------------|--------------------------------------|--------------|
| `driver-address`                | The socket file address for the COSI driver.                                                  | `unix:///var/lib/cosi/cosi.sock`     | Yes          |
| `driver-prefix`                 | The prefix for the COSI driver (e.g., `<prefix>.scality.com`).                                | `cosi`                               | No           |
| `driver-metrics-address`        | The address (hostname:port) for exposing Prometheus metrics.                                  | `:8080`                              | No           |
| `driver-metrics-path`           | The HTTP path for exposing metrics.                                                           | `/metrics`                           | No           |
| `driver-custom-metrics-prefix`  | The prefix for metrics collected by the COSI driver.                                          | `scality_cosi_driver`                | No           |
| `driver-otel-endpoint`          | The OpenTelemetry (OTEL) endpoint for exporting traces (if `driver-otel-stdout` is false).    | `""` (empty string disables tracing) | No           |
| `driver-otel-stdout`            | Enable OpenTelemetry trace export to stdout. Disables the OTEL endpoint if set to `true`.     | `false`                              | No           |
| `driver-otel-service-name`      | The service name reported in OpenTelemetry traces.                                            | `cosi.scality.com`                   | No           |

For Helm deployments, these parameters can be set in the [values.yaml](../helm/scality-cosi-driver/values.yaml) file or passed as flags during installation.

## Notes on OpenTelemetry Parameters

- **`driver-otel-endpoint`**:  
  Use this to specify an OTEL collector endpoint such as `otel-collector.local:4318`.  
  If `driver-otel-stdout` is set to `true`, this endpoint is ignored.

- **`driver-otel-stdout`**:  
  If set, trace data is printed to stdout in addition to any logging.  
  This is useful for local debugging but should generally be disabled in production.

- **`driver-otel-service-name`**:  
  Defines how the service is labeled in OTEL-based observability platforms (e.g., Jaeger).  

### Notes

- If driver-metrics-path does not end with `/`, it will automatically append `/`.
- Prometheus metrics are exposed for monitoring at the address and path specified.
- Generation of traces are disabled by default. To enable tracing, set `driver-otel-endpoint` to the desired OTEL collector endpoint or set `driver-otel-stdout` to `true` to print traces to stdout
