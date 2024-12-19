# COSI Driver Metrics Documentation

This document provides an overview of the Prometheus metrics exposed by the COSI driver. These metrics are designed to help open-source users monitor the performance and operations of the COSI driver. The metrics cover gRPC server calls, S3 API operations, and IAM API interactions.

## Metrics Overview

Metrics are exposed at the `/metrics` endpoint on the address configured via the `--metrics-address` flag (default: `:8080`). These metrics are Prometheus-compatible and can be used to create dashboards for observability.

---

## gRPC Metrics

The COSI driver exposes gRPC server metrics to monitor RPC activity.

| Metric Name                     | Description                                                | Labels                                     |
|---------------------------------|------------------------------------------------------------|--------------------------------------------|
| `grpc_server_started_total`     | Total number of RPCs started on the server.                | `grpc_method`, `grpc_service`, `grpc_type` |
| `grpc_server_handled_total`     | Total number of RPCs completed on the server.              | `grpc_method`, `grpc_service`, `grpc_code` |
| `grpc_server_msg_received_total`| Total number of messages received by the server.           | `grpc_method`, `grpc_service`             |
| `grpc_server_msg_sent_total`    | Total number of messages sent by the server.               | `grpc_method`, `grpc_service`             |
| `grpc_server_handling_seconds`  | Time taken for RPC calls to be handled by the server.       | `grpc_method`, `grpc_service`             |

### Example gRPC Methods

- Methods: `DriverCreateBucket`, `DriverDeleteBucket`, `DriverGetInfo`, `DriverGrantBucketAccess`, `DriverRevokeBucketAccess`
- Services: `cosi.v1alpha1.Provisioner`, `cosi.v1alpha1.Identity`

---

## S3 Metrics

These metrics track the total number of S3 requests and the duration of each request.

| Metric Name                      | Description                                                 | Labels            |
|----------------------------------|-------------------------------------------------------------|-------------------|
| `s3_requests_total`              | Total number of S3 requests, categorized by method and status. | `method`, `status` |
| `s3_request_duration_seconds`    | Duration of S3 requests in seconds, categorized by method and status. | `method`, `status` |

### Labels for S3

- Methods: `CreateBucket`, `DeleteBucket`
- Status: `success`, `error`

---

## IAM Metrics

These metrics track the total number of IAM requests and the duration of each request:

| Metric Name                      | Description                                                       | Labels                |
|----------------------------------|-------------------------------------------------------------------|-----------------------|
| `iam_requests_total`             | Total number of IAM requests, categorized by method and status.   | `method`, `status`   |
| `iam_request_duration_seconds`   | Duration of IAM requests in seconds, categorized by method and status. | `method`, `status`   |

### Labels for IAM

- Methods: `CreateUser`, `PutUserPolicy`, `CreateAccessKey`, `GetUser`, `DeleteUserPolicy`, `ListAccessKeys`, `DeleteAccessKey`, `DeleteUser`
- Status: `success`, `error`

---

## Example Metrics Output

### gRPC

```sh
grpc_server_started_total{grpc_method="DriverCreateBucket",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 5
grpc_server_started_total{grpc_method="DriverDeleteBucket",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 2
grpc_server_started_total{grpc_method="DriverGrantBucketAccess",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 3
grpc_server_started_total{grpc_method="DriverRevokeBucketAccess",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 1
grpc_server_started_total{grpc_method="DriverGetInfo",grpc_service="cosi.v1alpha1.Identity",grpc_type="unary"} 6

grpc_server_handled_total{grpc_method="DriverCreateBucket",grpc_service="cosi.v1alpha1.Provisioner",grpc_code="OK"} 5
grpc_server_handled_total{grpc_method="DriverDeleteBucket",grpc_service="cosi.v1alpha1.Provisioner",grpc_code="NotFound"} 2
```

### S3

```sh
s3_requests_total{method="CreateBucket",status="success"} 3
s3_requests_total{method="CreateBucket",status="error"} 1
s3_requests_total{method="DeleteBucket",status="success"} 4
s3_requests_total{method="DeleteBucket",status="error"} 2

s3_request_duration_seconds_bucket{method="CreateBucket",status="success",le="0.005"} 1
s3_request_duration_seconds_bucket{method="CreateBucket",status="success",le="0.01"} 2
s3_request_duration_seconds_bucket{method="CreateBucket",status="error",le="0.02"} 1
s3_request_duration_seconds_bucket{method="DeleteBucket",status="success",le="0.01"} 3
s3_request_duration_seconds_bucket{method="DeleteBucket",status="error",le="0.05"} 2
```

### IAM

```sh
iam_requests_total{method="CreateUser",status="success"} 4
iam_requests_total{method="CreateUser",status="error"} 1
iam_requests_total{method="PutUserPolicy",status="success"} 3
iam_requests_total{method="DeleteUserPolicy",status="error"} 2
iam_requests_total{method="CreateAccessKey",status="success"} 5
iam_requests_total{method="ListAccessKeys",status="success"} 6
iam_requests_total{method="DeleteAccessKey",status="error"} 2
iam_requests_total{method="DeleteUser",status="success"} 3
iam_requests_total{method="DeleteUser",status="error"} 1

iam_request_duration_seconds_bucket{method="CreateUser",status="success",le="0.005"} 3
iam_request_duration_seconds_bucket{method="CreateUser",status="error",le="0.01"} 1
iam_request_duration_seconds_bucket{method="DeleteUser",status="success",le="0.02"} 2
iam_request_duration_seconds_bucket{method="DeleteUser",status="error",le="0.05"} 1
iam_request_duration_seconds_bucket{method="ListAccessKeys",status="success",le="0.01"} 5
iam_request_duration_seconds_bucket{method="DeleteAccessKey",status="error",le="0.02"} 2
```

## Additional Resource

- [gRPC-Go Prometheus Metrics](https://github.com/grpc-ecosystem/go-grpc-prometheus)
- [Default Prometheus Metrics](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-subdirectories)
