# COSI Sidecar Error Handling Fix - Upstream Proposal

## Problem Statement

The current COSI sidecar has overly broad retry logic that only treats `codes.AlreadyExists` as non-retryable, causing infinite retry loops for legitimate client errors like `InvalidBucketName`, `PermissionDenied`, etc.

## Current Broken Logic

```go
// In bucket_listener.go - DriverCreateBucket
rsp, err := b.provisionerClient.DriverCreateBucket(ctx, req)
if err != nil {
    if status.Code(err) != codes.AlreadyExists {
        return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
    }
}
```

**Issue**: ALL errors except `AlreadyExists` are treated as retryable, including:
- `InvalidArgument` (bad bucket names) 
- `PermissionDenied` (access denied)
- `FailedPrecondition` (bucket not empty)
- `ResourceExhausted` (quota exceeded)

## Proposed Minimal Fix

### Principle: Only Internal Errors Should Retry

Only `codes.Internal` represents transient failures that could succeed on retry. All other errors are permanent failures that should not be retried.

```go
// isRetryableError determines if a gRPC error should be retried
func isRetryableError(err error) bool {
    code := status.Code(err)
    
    // Only internal errors (network issues, temporary backend failures) should retry
    // All client errors, permission errors, and precondition failures are permanent
    return code == codes.Internal
}

// Updated bucket creation logic
rsp, err := b.provisionerClient.DriverCreateBucket(ctx, req)
if err != nil {
    if !isRetryableError(err) {
        klog.V(3).ErrorS(err, "Non-retryable error from driver", 
            "bucket", bucket.ObjectMeta.Name, "errorCode", status.Code(err))
        return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
    }
    // For retryable errors (codes.Internal), return error to trigger controller retry
    klog.V(3).ErrorS(err, "Retryable error from driver - will retry with backoff",
        "bucket", bucket.ObjectMeta.Name, "errorCode", status.Code(err))
    return fmt.Errorf("failed to create bucket (retryable): %w", err)
}
```

## Error Code Classification

| gRPC Code | Should Retry? | Reasoning |
|-----------|---------------|-----------|
| `OK` | No | Success |
| `AlreadyExists` | No | Resource already exists (idempotent success) |
| `InvalidArgument` | No | Bad bucket name, invalid parameters - won't fix themselves |
| `PermissionDenied` | No | Access denied - credentials won't change |
| `FailedPrecondition` | No | Bucket not empty, state conflicts - require manual intervention |
| `ResourceExhausted` | No | Quota limits - require admin action |
| `NotFound` | No | For delete operations, this is success |
| `Unauthenticated` | No | Invalid credentials - won't fix themselves |
| `Unimplemented` | No | Driver doesn't support operation |
| `Internal` | **YES** | Transient failures, network issues, temporary backend problems |
| `Unavailable` | No* | Backend down - controller should know immediately |
| `DeadlineExceeded` | No* | Timeouts indicate real issues |
| `Unknown` | No* | Unknown errors likely permanent |

*Note: The current proposal is conservative. Future discussions could determine if some of these should be retryable.

## Files to Modify

### Primary Changes
- `sidecar/pkg/bucket/bucket_listener.go`:
  - Add `isRetryableError()` function
  - Update `Add()` method for bucket creation
  - Update `deleteBucketOp()` method for bucket deletion

### Secondary Files (if they exist)
- Any other listeners that call driver methods (BucketAccess operations)

## Impact

### Before Fix
- ❌ All non-`AlreadyExists` errors retry infinitely
- ❌ Invalid bucket names retry forever
- ❌ Permission denied retries forever  
- ❌ Resource exhaustion from exponential backoff

### After Fix  
- ✅ Only genuine transient failures retry
- ✅ Client errors fail immediately with clear messages
- ✅ Permission errors fail immediately
- ✅ State conflicts fail immediately
- ✅ Proper resource utilization

## Backward Compatibility

This change is **backward compatible**:
- Successful operations remain unchanged
- `AlreadyExists` handling remains unchanged
- Only improves error handling for problematic cases

## Testing

Test cases should cover:
1. `InvalidArgument` errors (bad bucket names) - should NOT retry
2. `PermissionDenied` errors - should NOT retry
3. `FailedPrecondition` errors (bucket not empty) - should NOT retry
4. `Internal` errors - SHOULD retry with backoff
5. `AlreadyExists` - should NOT retry (existing behavior)

## Implementation

This fix can be implemented as:
1. **Immediate patch** for critical environments
2. **Upstream contribution** to kubernetes-sigs/container-object-storage-interface
3. **Backward-portable** to existing COSI versions 