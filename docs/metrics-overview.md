# COSI Driver Metrics Documentation

This document provides an overview of the Prometheus metrics exposed by the COSI driver. These metrics are designed to help open-source users monitor the performance and operations of the COSI driver. The metrics cover gRPC server calls.

## Metrics Overview

Metrics are exposed at the `/metrics` endpoint on the address configured via the `--metrics-address` flag (default: `:8080`). These metrics are Prometheus-compatible and can be used to create dashboards for observability.

---

## gRPC Default Metrics

The COSI driver exposes default gRPC server metrics to monitor RPC activity.

| Metric Name                     | Description                                                | Labels                                     |
|---------------------------------|------------------------------------------------------------|--------------------------------------------|
| `grpc_server_started_total`     | Total number of RPCs started on the server.                | `grpc_method`, `grpc_service`, `grpc_type` |
| `grpc_server_handled_total`     | Total number of RPCs completed on the server.              | `grpc_method`, `grpc_service`, `grpc_code` |
| `grpc_server_msg_received_total`| Total number of messages received by the server.           | `grpc_method`, `grpc_service`              |
| `grpc_server_msg_sent_total`    | Total number of messages sent by the server.               | `grpc_method`, `grpc_service`              |
| `grpc_server_handling_seconds`  | Time taken for RPC calls to be handled by the server.      | `grpc_method`, `grpc_service`              |

### Example gRPC Methods

- Methods: `DriverCreateBucket`, `DriverDeleteBucket`, `DriverGetInfo`, `DriverGrantBucketAccess`, `DriverRevokeBucketAccess`
- Services: `cosi.v1alpha1.Provisioner`, `cosi.v1alpha1.Identity`

```sh
grpc_server_started_total{grpc_method="DriverGetInfo",grpc_service="cosi.v1alpha1.Identity",grpc_type="unary"} 2
```

---
## IAM Operation Metrics

The COSI driver collects metrics for IAM operations performed via the AWS IAM API. These metrics help track the number and duration of IAM-related operations, enabling better monitoring and observability of IAM activity.

Status Labels

| Label     | Description                                     |
|-----------|-------------------------------------------------|
| `success` | Indicates the operation completed successfully. |
| `error`   | Indicates the operation failed.                 |

### Key IAM Metrics

| Metric Name                                        | Description                                                | Labels            | Example Values                  |
|---------------------------------------------------|------------------------------------------------------------|-------------------|----------------------------------|
| `scality_cosi_driver_iam_request_duration_seconds`| Histogram of IAM request durations in seconds.             | `action`, `status`| `CreateUser`, `success`         |
| `scality_cosi_driver_iam_requests_total`          | Total number of IAM requests categorized by action and status. | `action`, `status`| `CreateAccessKey`, `success`    |

### IAM Operations

| IAM Operation          | Description                                                          |
|-------------------------|---------------------------------------------------------------------|
| `CreateUser`           | Creates an IAM user with the specified username.                     |
| `CreateAccessKey`      | Generates access keys for a specific IAM user.                       |
| `PutUserPolicy`        | Attaches an inline S3 wildcard policy to a user for bucket access.   |
| `GetUser`              | Retrieves details about an IAM user.                                 |
| `ListAccessKeys`       | Lists all access keys associated with an IAM user.                   |
| `DeleteAccessKey`      | Deletes a specific access key associated with an IAM user.           |
| `DeleteUserPolicy`     | Deletes an inline policy associated with an IAM user.                |
| `DeleteUser`           | Deletes an IAM user.                                                 |

### Example IAM Metrics Output

Duration of IAM requests in seconds

```sh
scality_cosi_driver_iam_request_duration_seconds_bucket{action="CreateUser",status="success",le="0.01"} 3
scality_cosi_driver_iam_request_duration_seconds_bucket{action="CreateUser",status="success",le="0.025"} 4
scality_cosi_driver_iam_request_duration_seconds_sum{action="CreateUser",status="success"} 0.014
scality_cosi_driver_iam_request_duration_seconds_count{action="CreateUser",status="success"} 4
```

Total number of IAM requests

```sh
scality_cosi_driver_iam_requests_total{action="CreateUser",status="success"} 4
scality_cosi_driver_iam_requests_total{action="DeleteAccessKey",status="error"} 1
```

### Example IAM Workflow

#### Creating Bucket Access

1. Create an IAM user (`CreateUser`).
2. Attach an inline policy for bucket **access** (`PutUserPolicy`).
3. Generate access keys for the IAM user (`CreateAccessKey`).

#### Revoking Bucket Access

1. Verify the IAM user exists (`GetUser`).
2. Delete inline policies (`DeleteUserPolicy`).
3. Delete all associated access keys (`DeleteAccessKey`).
4. Delete the IAM user (`DeleteUser`).

---

## S3 Operation Metrics

The COSI driver collects metrics for S3 bucket operations performed via the AWS S3 API. These metrics help monitor bucket-related operations and their durations.

### Status Labels

```sh
| Label     | Description                                     |
|-----------|-------------------------------------------------|
| `success` | Indicates the operation completed successfully. |
| `error`   | Indicates the operation failed.                 |
```

### Key S3 Metrics

| Metric Name                                        | Description                                                | Labels            | Example Values                  |
|---------------------------------------------------|------------------------------------------------------------|-------------------|----------------------------------|
| `scality_cosi_driver_s3_request_duration_seconds` | Histogram of S3 request durations in seconds.             | `action`, `status`| `CreateBucket`, `success`       |
| `scality_cosi_driver_s3_requests_total`           | Total number of S3 requests categorized by action and status. | `action`, `status`| `DeleteBucket`, `success`       |

### S3 Operations

| S3 Operation        | Description                                                              |
|---------------------|--------------------------------------------------------------------------|
| `CreateBucket`      | Creates a new S3 bucket in the specified region.                         |
| `DeleteBucket`      | Deletes an existing S3 bucket. (only empty bucket deletion is supported) |

### Example S3 Metrics Output

Duration of S3 requests in seconds

```sh
scality_cosi_driver_s3_request_duration_seconds_bucket{action="CreateBucket",status="success",le="0.01"} 1
scality_cosi_driver_s3_request_duration_seconds_bucket{action="CreateBucket",status="success",le="0.05"} 2
scality_cosi_driver_s3_request_duration_seconds_sum{action="CreateBucket",status="success"} 0.04
scality_cosi_driver_s3_request_duration_seconds_count{action="CreateBucket",status="success"} 2
```

Total number of S3 requests

```sh
scality_cosi_driver_s3_requests_total{action="CreateBucket",status="success"} 2
scality_cosi_driver_s3_requests_total{action="DeleteBucket",status="success"} 1
```

### Example S3 Workflow

#### Creating a Bucket

1. Specify the bucket name and region.
2. Use the `CreateBucket` operation to create the bucket.
3. Configure bucket properties (e.g., policies, versioning) if needed.

#### Deleting a Bucket

1. Verify the bucket exists.
2. Use the `DeleteBucket` operation to delete the bucket. Only empty bucket deletion is supported.

## Additional Resource

- [gRPC-Go Prometheus Metrics](https://github.com/grpc-ecosystem/go-grpc-middleware)
- [Default Prometheus Metrics](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-subdirectories)
