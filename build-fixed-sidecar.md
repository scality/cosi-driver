# Building Fixed COSI Sidecar - Immediate Solution

## Quick Fix Implementation

### Step 1: Clone and Patch COSI Repository

```bash
# Clone the main COSI repository
git clone https://github.com/kubernetes-sigs/container-object-storage-interface.git
cd container-object-storage-interface

# Create a new branch for the fix
git checkout -b fix/error-handling-retry-logic

# Apply the patch (you'll need to create the patch file from upstream-sidecar-patch.patch)
# For now, manually edit sidecar/pkg/bucket/bucket_listener.go
```

### Step 2: Modify the bucket_listener.go

Edit `sidecar/pkg/bucket/bucket_listener.go` and add:

```go
// Add after existing imports
// isRetryableError determines if a gRPC error should trigger a retry.
// Only codes.Internal errors are considered retryable as they represent
// transient failures (network issues, temporary backend problems).
// All other error codes represent permanent failures that won't be resolved by retrying.
func isRetryableError(err error) bool {
    code := status.Code(err)
    // Only internal errors should retry - all others are permanent failures
    return code == codes.Internal
}
```

Replace the existing error handling in `Add()` method around line 141:

```go
// OLD CODE:
if status.Code(err) != codes.AlreadyExists {
    return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
}

// NEW CODE:
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
```

And similar changes for delete operations.

### Step 3: Build the Fixed Sidecar Image

```bash
# Build the sidecar
cd sidecar
make build

# Build Docker image (update the tag as needed)
docker build -t scality/objectstorage-sidecar:fixed-v0.1.0 .

# Push to your registry
docker push scality/objectstorage-sidecar:fixed-v0.1.0
```

### Step 4: Update Your Deployment

Update your Kustomize deployment to use the fixed image:

```yaml
# In kustomize/overlays/production/sidecar-image-patch.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cosi-driver
spec:
  template:
    spec:
      containers:
      - name: objectstorage-provisioner-sidecar
        image: scality/objectstorage-sidecar:fixed-v0.1.0
```

### Step 5: Deploy and Test

```bash
kubectl apply -k kustomize/overlays/production
```

## Expected Behavior After Fix

### ✅ With Fixed Sidecar
```
# Driver logs (unchanged)
E0616 20:15:22.778377 "Invalid bucket name - will not retry" 
rpc error: code = InvalidArgument desc = Invalid bucket name

# NEW Sidecar logs  
I0616 20:15:22.779000 "Non-retryable error from driver" bucket="long-bucket-name" errorCode="InvalidArgument"
E0616 20:15:22.779100 "Bucket creation failed permanently" error="failed to create bucket: rpc error: code = InvalidArgument"

# Controller logs
I0616 20:15:22.780000 "Bucket marked as failed - no retry"
```

### ❌ Current Broken Behavior
```
# Infinite retry loop every 1s, 2s, 4s, 8s, 16s, 32s, 64s...
# Resource exhaustion
# No clear failure indication
```

## Verification

Test with the same long bucket name that was causing infinite retries:
- ✅ Should fail immediately with clear error message
- ✅ No retry attempts
- ✅ Bucket marked as failed
- ✅ Controller moves on to next task 