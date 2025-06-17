# COSI Sidecar Infinite Retry Bug - GitHub Issue

**Copy this entire content for the GitHub issue:**

---

## 🐛 **Bug Report: Sidecar retries all non-AlreadyExists errors infinitely**

### **Problem Description**

The COSI sidecar has overly broad retry logic that only treats `codes.AlreadyExists` as non-retryable, causing infinite retry loops for legitimate permanent errors like `InvalidArgument`, `PermissionDenied`, `FailedPrecondition`, etc.

### **Current Broken Behavior**

```go
// In sidecar/pkg/bucket/bucket_listener.go around line 141
rsp, err := b.provisionerClient.DriverCreateBucket(ctx, req)
if err != nil {
    if status.Code(err) != codes.AlreadyExists {
        return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
    }
}
```

**❌ This means ALL errors except `AlreadyExists` trigger infinite retries:**
- `InvalidArgument` (bad bucket names, malformed parameters)
- `PermissionDenied` (access denied, insufficient permissions) 
- `FailedPrecondition` (bucket not empty, state conflicts)
- `ResourceExhausted` (quota limits exceeded)
- `Unauthenticated` (invalid credentials)
- `Unimplemented` (unsupported operations)

### **Real-World Impact**

**Scality COSI Driver (COSI-86)**: Invalid bucket names cause infinite retry loops despite the driver correctly returning `codes.InvalidArgument` with "will not retry" logs.

**Example logs:**
```
# Driver (working correctly)
E0616 20:15:22.778377 "Invalid bucket name - will not retry" 
bucketName="testlongnameforbucketgenerationerrorisexpectedtest-bucketclass3341a3a8"

# Sidecar (broken - keeps retrying)
I0616 20:15:22.881949 "Processing DriverCreateBucket request" 
I0616 20:15:23.094342 "Processing DriverCreateBucket request"
I0616 20:15:24.301823 "Processing DriverCreateBucket request"
# ... continues infinitely with exponential backoff
```

### **Impact on Users**

1. **🔥 Resource Exhaustion**: Exponential backoff (1s, 2s, 4s, 8s, 16s, 32s, 64s...) consumes cluster CPU/memory
2. **❌ Poor User Experience**: No clear failure indication - buckets stuck in "pending" forever
3. **🚨 Operational Issues**: Infinite retry loops mask real configuration problems
4. **💸 Cost Impact**: Wasted compute resources on impossible operations

### **Proposed Solution**

**Conservative Approach**: Only `codes.Internal` errors should trigger retries.

**Rationale**: Only `codes.Internal` represents genuine transient failures (network blips, temporary backend issues). All other gRPC error codes represent permanent failures that won't be resolved by retrying.

```go
// isRetryableError determines if a gRPC error should trigger a retry
func isRetryableError(err error) bool {
    code := status.Code(err)
    // Only internal errors should retry - all others are permanent failures
    return code == codes.Internal
}

// Updated error handling
rsp, err := b.provisionerClient.DriverCreateBucket(ctx, req)
if err != nil {
    if !isRetryableError(err) {
        // Non-retryable error: fail immediately with clear error message
        klog.V(3).ErrorS(err, "Non-retryable error from driver", 
            "bucket", bucket.ObjectMeta.Name, "errorCode", status.Code(err))
        return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
    }
    // Retryable error (codes.Internal): return error to trigger controller retry
    klog.V(3).ErrorS(err, "Retryable error from driver - will retry with backoff",
        "bucket", bucket.ObjectMeta.Name, "errorCode", status.Code(err))
    return fmt.Errorf("failed to create bucket (retryable): %w", err)
}
```

### **Error Code Classification**

| gRPC Code | Should Retry? | Reasoning |
|-----------|---------------|-----------|
| `Internal` | ✅ **YES** | Network issues, temporary backend failures |
| `InvalidArgument` | ❌ No | Bad bucket names, invalid parameters won't fix themselves |
| `PermissionDenied` | ❌ No | Access denied won't change without admin intervention |
| `FailedPrecondition` | ❌ No | State conflicts (bucket not empty) require manual action |
| `ResourceExhausted` | ❌ No | Quota limits require admin action to increase |
| `AlreadyExists` | ❌ No | Success for idempotent operations |
| `NotFound` | ❌ No | For delete operations, this is success |
| `Unauthenticated` | ❌ No | Invalid credentials won't fix themselves |
| `Unimplemented` | ❌ No | Driver doesn't support operation |
| `Unavailable` | ❌ No | Backend down - controller should know immediately |
| `DeadlineExceeded` | ❌ No | Timeout indicates real issues |
| `Unknown` | ❌ No | Unknown errors likely permanent |

### **Files Affected**

- `sidecar/pkg/bucket/bucket_listener.go` (primary fix)
- Any other listeners with similar error handling patterns

### **Backward Compatibility**

✅ **Fully backward compatible:**
- Successful operations unchanged
- `AlreadyExists` handling unchanged  
- Only improves problematic infinite retry cases
- No API changes

### **Testing Plan**

- Unit tests for `isRetryableError()` function with all error codes
- Integration tests verifying immediate failure for permanent errors
- E2E tests with mock driver returning various error codes
- Regression tests ensuring existing behavior preserved

### **Expected Benefits**

✅ **Immediate failure for permanent errors** (InvalidArgument, PermissionDenied, etc.)  
✅ **Resource efficiency** - no more infinite retry loops  
✅ **Better observability** - clear error messages instead of endless retries  
✅ **Improved user experience** - fast failure feedback  
✅ **Benefits entire COSI ecosystem** - all drivers will benefit  

### **Reproduction Steps**

1. Deploy any COSI driver
2. Create BucketClaim with invalid bucket name (>63 chars or invalid format)
3. Driver correctly returns `codes.InvalidArgument`
4. Observe sidecar retrying infinitely instead of failing immediately

### **Environment**

- **COSI Version**: v0.1.0+ (affects all versions)
- **Kubernetes**: Any version
- **Driver**: Any COSI driver (Scality, AWS, GCS, etc.)

---

**Labels to add:** `kind/bug`, `area/sidecar`, `priority/high`

**This issue affects all COSI deployments and represents a fundamental flaw in the retry logic that wastes cluster resources and provides poor user experience.** 