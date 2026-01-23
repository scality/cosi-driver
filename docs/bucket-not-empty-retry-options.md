# BucketNotEmpty Infinite Retry Issue - Options Analysis

## Problem

When deleting a BucketClaim linked to a non-empty bucket, the COSI controller retries the deletion indefinitely with exponential backoff.

**Current behavior:**
1. User deletes BucketClaim
2. COSI controller calls driver's `DriverDeleteBucket`
3. S3 returns `BucketNotEmpty` error
4. Driver returns gRPC `FailedPrecondition` status
5. COSI controller retries (infinite loop until bucket is emptied)

## Root Cause

The COSI sidecar determines retry behavior in [`rpcErrorIsRetryable()`](https://github.com/kubernetes-sigs/container-object-storage-interface/blob/main/sidecar/pkg/reconciler/driver.go):

```go
func rpcErrorIsRetryable(c codes.Code) bool {
    switch c {
    case codes.InvalidArgument:
        return false  // Non-retryable
    case codes.AlreadyExists:
        return false  // Non-retryable
    case codes.Unimplemented:
        return false  // Non-retryable
    default:
        return true   // FailedPrecondition falls here → RETRIED
    }
}
```

`FailedPrecondition` is not explicitly handled, so it defaults to **retryable**.

---

## Options

### Option 1: Change Error Code to `InvalidArgument`

Change the driver to return `InvalidArgument` instead of `FailedPrecondition` for non-empty buckets.

| Pros | Cons |
|------|------|
| Stops infinite retries immediately | Semantically incorrect per gRPC spec |
| No upstream dependencies | `InvalidArgument` implies bad request (misleading) |
| Quick to implement | |

---

### Option 2: Contribute Fix to Upstream COSI

Submit a PR to [kubernetes-sigs/container-object-storage-interface](https://github.com/kubernetes-sigs/container-object-storage-interface) adding `FailedPrecondition` to the non-retryable list.

| Pros | Cons |
|------|------|
| Semantically correct solution | Requires upstream approval and release cycle |
| Benefits all COSI drivers | We'd need to wait or use patched version |
| Aligns with gRPC best practices | |

**Note:** The gRPC spec explicitly states `FailedPrecondition` should not be retried:
> "Use FailedPrecondition if the client should not retry until the system state has been explicitly fixed."

---

### Option 3: Keep Current Behavior + Documentation

Keep returning `FailedPrecondition` and document the expected behavior.

| Pros | Cons |
|------|------|
| Semantically correct | Infinite retries fill logs |
| No code changes | Confusing UX for users |
| Bucket deletes when eventually emptied | |

---

### Option 4: Combination Approach

- **Short-term:** Use `InvalidArgument` to stop retries now
- **Long-term:** File upstream issue / contribute PR to fix COSI controller

| Pros | Cons |
|------|------|
| Immediate relief for users | Two-phase effort |
| Correct long-term solution | Need to revert driver change after upstream fix |

---

## Summary Table

| Option | Stops Retries | Semantically Correct | Upstream Dependency | Effort |
|--------|---------------|---------------------|---------------------|--------|
| 1. InvalidArgument | Yes | No | None | Low |
| 2. Upstream PR | Yes | Yes | High | Medium |
| 3. Keep + Document | No | Yes | None | Low |
| 4. Both | Yes | Eventually | Medium | Medium |

---

## References

- [COSI Controller Source - rpcErrorIsRetryable()](https://github.com/kubernetes-sigs/container-object-storage-interface/blob/main/sidecar/pkg/reconciler/driver.go)
- [gRPC Status Codes Guide](https://grpc.io/docs/guides/status-codes/)
- Current driver implementation: `pkg/osperrors/s3_errors.go`
