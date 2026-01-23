# Frequently Asked Questions

## IAM User Naming

### Why are IAM users named "ba-UUID" instead of something meaningful?

When the Scality COSI Driver creates an IAM user for bucket access, the username follows the format `ba-<UUID>` (e.g., `ba-7f3d2a1e-9c4b-4f8a-b2e1-6d5c4a3b2e1f`).

**This naming convention is defined by the COSI specification, not by the Scality driver.**

| Component | Responsibility |
|-----------|----------------|
| COSI Controller (upstream Kubernetes project) | Generates the `ba-<UUID>` name from the BucketAccess resource |
| Scality COSI Driver | Receives this name and uses it as-is for the IAM username |

The "ba-" prefix stands for "**B**ucket**A**ccess" and the UUID is auto-generated when a BucketAccess resource is created in Kubernetes.

### Why doesn't the IAM username reference the BucketAccessClass?

The [COSI specification](https://github.com/kubernetes-sigs/container-object-storage-interface-spec) does not include BucketAccessClass information in the driver request. When granting bucket access, the driver only receives:

- `bucket_id` - which bucket to grant access to
- `name` - the BucketAccess resource name (ba-UUID)
- `authentication_type` - KEY or IAM
- `parameters` - opaque key-value pairs from BucketAccessClass

The BucketAccessClass name itself is not passed to the driver.

### Why does the driver use the name as-is instead of generating its own?

The COSI specification requires a **round-trip identifier** for access management:

1. **Grant Access**: Controller sends `name` → Driver creates IAM user → Driver returns `account_id`
2. **Revoke Access**: Controller sends `account_id` → Driver deletes IAM user

The driver returns the same `name` it received as the `account_id`. This allows the COSI controller to later identify which IAM user to delete when revoking access. If the driver modified or generated a different name, the revoke operation would not know which IAM user to delete.

### How can I identify which BucketAccess created an IAM user?

You can correlate IAM users to Kubernetes resources using kubectl:

```bash
# List all BucketAccess resources
kubectl get bucketaccess

# Find the BucketAccess with the matching UUID
kubectl get bucketaccess -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.accountID}{"\n"}{end}'
```

The `status.accountID` field in the BucketAccess resource contains the IAM username (`ba-<UUID>`).

### Can the naming be customized?

Currently, the naming convention cannot be customized without changes to the upstream COSI specification or driver enhancements.

| Option | Feasibility | Notes |
|--------|-------------|-------|
| Add metadata to `parameters` | Medium | BucketAccessClass parameters are available during **grant** but NOT during **revoke**. The driver would need to persist this metadata (e.g., in IAM user tags) during grant to retrieve it during revoke. |
| Upstream COSI spec change | Hard | Would require proposing a change to the Kubernetes COSI specification |

**Important limitation:** The COSI specification does not pass BucketAccessClass parameters to the `DriverRevokeBucketAccess` call. The revoke operation only receives:
- `bucket_id` - the bucket name
- `account_id` - the IAM username (ba-UUID)

The driver currently fetches connection parameters from the **Bucket** object (BucketClass parameters), not from BucketAccessClass. Any custom naming solution would require the driver to store metadata during grant (e.g., as IAM user tags) and retrieve it during revoke.

---

## Bucket Deletion

### Why does deleting a BucketClaim for a non-empty bucket retry indefinitely?

When you delete a BucketClaim linked to a bucket that still contains objects, the COSI controller retries the deletion indefinitely with exponential backoff.

**This is expected behavior** based on how the COSI controller handles errors, though it can be confusing.

#### What happens

1. You delete the BucketClaim
2. COSI controller calls the driver's `DriverDeleteBucket`
3. Driver attempts to delete the bucket in S3
4. S3 returns `BucketNotEmpty` error
5. Driver translates this to gRPC `FailedPrecondition` status
6. COSI controller receives the error and retries with exponential backoff
7. Steps 2-6 repeat indefinitely until the bucket is emptied

#### Why FailedPrecondition?

The driver returns `FailedPrecondition` because this is the semantically correct gRPC status code. According to [gRPC conventions](https://grpc.io/docs/guides/status-codes/):

> "Use `FailedPrecondition` if the client should not retry until the system state has been explicitly fixed. E.g., if an 'rmdir' fails because the directory is non-empty."

However, the [COSI specification](https://github.com/kubernetes-sigs/container-object-storage-interface-spec) does not explicitly define which error codes are terminal vs retriable, so the controller retries all errors.

#### How to resolve

Empty the bucket before deleting the BucketClaim:

```bash
# Using AWS CLI
aws s3 rm s3://<bucket-name> --recursive --endpoint-url <S3_ENDPOINT>

# Then delete the BucketClaim
kubectl delete bucketclaim <bucketclaim-name>
```

#### How to identify stuck deletions

Check the driver logs for repeated `BucketNotEmpty` errors:

```bash
kubectl logs -n container-object-storage-system -l app.kubernetes.io/name=scality-cosi-driver | grep "not empty"
```

You can also monitor the `scality_cosi_driver_s3_requests_total` metric with `action="DeleteBucket"` and `status="BucketNotEmpty"` for repeated failures.

#### Can this behavior be changed?

| Option | Status |
|--------|--------|
| Empty bucket automatically | Not implemented - intentional safety measure to prevent data loss |
| Stop retrying after N attempts | Requires upstream COSI controller change |
| Return terminal error code | Would be semantically incorrect per gRPC conventions |

This is a known limitation of the current COSI specification. Consider filing an issue with the [upstream COSI project](https://github.com/kubernetes-sigs/container-object-storage-interface) if this significantly impacts your workflows.

---

## Monitoring and Alerting

### How do I access the metrics exposed by the COSI driver?

The Scality COSI Driver exposes Prometheus-compatible metrics via an HTTP endpoint.

#### Endpoint Details

| Setting | Default Value |
|---------|---------------|
| Port | `8080` |
| Path | `/metrics` |
| Address | `0.0.0.0:8080` |
| Metrics Prefix | `scality_cosi_driver` |

The driver automatically starts a metrics server when deployed. A Kubernetes Service named `scality-cosi-driver-metrics` is created to expose this endpoint within the cluster.

#### Accessing Metrics Manually

**Port-forward to the driver pod:**

```bash
# Port-forward to access metrics locally
kubectl port-forward -n container-object-storage-system \
  deployment/scality-cosi-driver 8080:8080

# Query metrics
curl http://localhost:8080/metrics
```

**Via the metrics service:**

```bash
# Port-forward via the service
kubectl port-forward -n container-object-storage-system \
  svc/scality-cosi-driver-metrics 8080:8080

# Query metrics
curl http://localhost:8080/metrics
```

#### Configuring Prometheus to Scrape Metrics

**Option 1: Prometheus ServiceMonitor (for prometheus-operator)**

Create a ServiceMonitor resource to automatically discover and scrape the driver:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: scality-cosi-driver
  namespace: container-object-storage-system
  labels:
    app.kubernetes.io/name: scality-cosi-driver
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: scality-cosi-driver
  endpoints:
    - port: "8080"
      path: /metrics
      interval: 30s
```

**Option 2: Static Prometheus scrape config**

Add this job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'scality-cosi-driver'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - container-object-storage-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: scality-cosi-driver-metrics
```

**Option 3: Pod annotation-based discovery**

If your Prometheus is configured to scrape annotated pods, add these annotations to the driver deployment:

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

#### Customizing Metrics Configuration

**Helm values:**

```yaml
metrics:
  enabled: true
  port: 8080
  prefix: "scality_cosi_driver"
  address: "0.0.0.0:8080"
  path: "/metrics"
```

**Driver CLI flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--driver-metrics-address` | `:8080` | Address to expose metrics |
| `--driver-metrics-path` | `/metrics` | HTTP path for metrics endpoint |
| `--driver-custom-metrics-prefix` | `scality_cosi_driver` | Prefix for all metric names |

---

### How should I monitor bucket operation failures?

The Scality COSI Driver provides multiple observability mechanisms to monitor and alert on failures during bucket operations (create, delete, grant/revoke access).

#### Prometheus Metrics

The driver exposes Prometheus metrics on port `8080` at the `/metrics` endpoint. These metrics are automatically available when the driver is deployed.

**Available Metrics:**

| Metric | Labels | Description |
|--------|--------|-------------|
| `scality_cosi_driver_s3_requests_total` | `action`, `status` | Total count of S3 operations |
| `scality_cosi_driver_s3_request_duration_seconds` | `action`, `status` | Latency histogram for S3 operations |
| `scality_cosi_driver_iam_requests_total` | `action`, `status` | Total count of IAM operations |
| `scality_cosi_driver_iam_request_duration_seconds` | `action`, `status` | Latency histogram for IAM operations |

The `action` label indicates the operation type (e.g., `CreateBucket`, `DeleteBucket`, `HeadBucket`).
The `status` label indicates the outcome (e.g., `success`, `BucketNotEmpty`, `BucketAlreadyExists`).

**Example metric output:**

```
# HELP scality_cosi_driver_s3_requests_total Total number of S3 requests, categorized by action and status.
# TYPE scality_cosi_driver_s3_requests_total counter
scality_cosi_driver_s3_requests_total{action="CreateBucket",status="success"} 5
scality_cosi_driver_s3_requests_total{action="DeleteBucket",status="BucketNotEmpty"} 12
scality_cosi_driver_s3_requests_total{action="HeadBucket",status="success"} 10
```

**Recommended Alerting Rules:**

```yaml
groups:
  - name: cosi-driver-alerts
    rules:
      # Alert when bucket operation failure rate exceeds 10%
      - alert: COSIBucketOperationFailureRate
        expr: |
          sum(rate(scality_cosi_driver_s3_requests_total{status!="success"}[5m]))
          / sum(rate(scality_cosi_driver_s3_requests_total[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High COSI bucket operation failure rate"
          description: "More than 10% of bucket operations are failing."

      # Alert when bucket deletion is blocked due to non-empty bucket
      - alert: COSIBucketDeletionBlocked
        expr: |
          increase(scality_cosi_driver_s3_requests_total{action="DeleteBucket",status="BucketNotEmpty"}[1h]) > 10
        labels:
          severity: warning
        annotations:
          summary: "Bucket deletion blocked - bucket contains objects"
          description: "A bucket cannot be deleted because it is not empty. Empty the bucket or remove the deletion request."

      # Alert on IAM credential operation failures
      - alert: COSIIAMOperationFailures
        expr: |
          increase(scality_cosi_driver_iam_requests_total{status!="success"}[10m]) > 5
        labels:
          severity: warning
        annotations:
          summary: "IAM credential operations failing"
          description: "Multiple IAM operations have failed. Check driver logs for details."
```

#### Kubernetes Object Status

The COSI controller maintains status conditions on `Bucket` and `BucketClaim` objects. Monitor these for operation state:

```bash
# List all buckets and their status
kubectl get buckets -A

# List all bucket claims and their status
kubectl get bucketclaims -A

# Get detailed status for a specific bucket
kubectl describe bucket <bucket-name>

# Get detailed status for a specific bucket claim
kubectl describe bucketclaim <claim-name> -n <namespace>
```

Failed operations will show error conditions on these objects with descriptive messages.

#### Log Monitoring

The driver logs operational details using structured logging. For log aggregation systems (Loki, Elasticsearch, Splunk), monitor for these patterns:

| Log Pattern | Meaning |
|-------------|---------|
| `"BucketNotEmpty"` | Bucket deletion blocked - bucket contains objects |
| `"BucketAlreadyOwnedByYou"` | Bucket creation skipped - already exists |
| `"NoSuchBucket"` | Operation on non-existent bucket |
| `"error"` + `DriverCreateBucket` | Bucket creation failure |
| `"error"` + `DriverDeleteBucket` | Bucket deletion failure |

To stream driver logs:

```bash
kubectl logs -f deployment/scality-cosi-driver \
  -n container-object-storage-system
```

#### Grafana Dashboard

For visualization, create a Grafana dashboard with these panels:

| Panel | PromQL Query |
|-------|--------------|
| Request Rate by Operation | `sum(rate(scality_cosi_driver_s3_requests_total[5m])) by (action)` |
| Error Rate by Status | `sum(rate(scality_cosi_driver_s3_requests_total{status!="success"}[5m])) by (status)` |
| Success Ratio | `sum(rate(scality_cosi_driver_s3_requests_total{status="success"}[5m])) / sum(rate(scality_cosi_driver_s3_requests_total[5m]))` |
| P99 Latency by Operation | `histogram_quantile(0.99, sum(rate(scality_cosi_driver_s3_request_duration_seconds_bucket[5m])) by (le, action))` |

#### Summary

| Method | Best For |
|--------|----------|
| Prometheus metrics | Automated alerting, trend analysis, SLO tracking |
| Kubernetes object status | Real-time operational visibility, debugging specific failures |
| Log monitoring | Root cause analysis, detailed error context |
| Grafana dashboards | Operational overview, capacity planning |

We recommend implementing all three layers for comprehensive observability of the COSI driver in production environments.

---

## Further Reading

- [COSI Specification](https://github.com/kubernetes-sigs/container-object-storage-interface-spec)
- [Usage Guide](./Usage.md)
- [Driver Parameters](./driver-params.md)
