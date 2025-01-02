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

## Additional Resource

- [gRPC-Go Prometheus Metrics](https://github.com/grpc-ecosystem/go-grpc-prometheus)
- [Default Prometheus Metrics](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-subdirectories)
