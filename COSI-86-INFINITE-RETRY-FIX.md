# COSI-86: Infinite Retry Loop - Root Cause Analysis & Complete Fix

## 🚨 **URGENT: The Real Problem**

**TL;DR**: The infinite retry issue in COSI-86 is **NOT in our driver code** - it's in the **upstream COSI sidecar**. Our driver properly returns `codes.InvalidArgument`, but the sidecar only treats `codes.AlreadyExists` as non-retryable!

## 🔍 **Root Cause Analysis**

### **What We Initially Thought**
We thought our driver was returning `codes.Internal` for all errors, causing infinite retries.

### **What We Fixed First** ✅
We enhanced our driver's error handling to return proper gRPC error codes:
- `codes.InvalidArgument` for client-side errors (InvalidBucketName, etc.)
- `codes.PermissionDenied` for authorization errors
- `codes.FailedPrecondition` for state conflicts (BucketNotEmpty)
- `codes.ResourceExhausted` for quota limits
- `codes.NotFound` for successful idempotent deletions

### **The Real Problem** ❌
The COSI sidecar has **hardcoded logic** that only treats `codes.AlreadyExists` as non-retryable:

```go
// In the upstream COSI sidecar:
rsp, err := b.provisionerClient.DriverCreateBucket(ctx, req)
if err != nil {
    if status.Code(err) != codes.AlreadyExists {  // ⬅️ ONLY AlreadyExists!
        return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
    }
}
```

**ALL other error codes (including our properly returned `InvalidArgument`, `PermissionDenied`, etc.) are treated as retryable errors!**

## 📋 **Evidence**

### **Driver Logs** (Our Fix Working) ✅
```
E0616 20:15:22.778377 "Invalid bucket name - will not retry" 
E0616 20:15:22.778377 rpc error: code = InvalidArgument desc = Invalid bucket name: testlongnameforbucketgenerationerrorisexpectedtest-bucketclass3341a3a8-4132-40f5-831f-99709a74eee0
```

### **Sidecar Logs** (Still Retrying) ❌
```
Message: "failed to create bucket: rpc error: code = InvalidArgument desc = Invalid bucket name: testlongnameforbucketgenerationerrorisexpectedtest-bucketclass3341a3a8-4132-40f5-831f-99709a74eee0"
```

The sidecar receives our `InvalidArgument` error but still retries because it's not `AlreadyExists`.

## 🎯 **The Complete Solution**

### **Option 1: Fix the COSI Sidecar** (Recommended)

We need to modify the COSI sidecar to properly handle non-retryable error codes.

#### **Key Changes Needed**
1. **Create an `isRetryableError()` function** that classifies gRPC error codes:
   ```go
   func isRetryableError(err error) bool {
       code := status.Code(err)
       switch code {
       case codes.OK, codes.AlreadyExists, codes.InvalidArgument, 
            codes.PermissionDenied, codes.FailedPrecondition, 
            codes.ResourceExhausted, codes.NotFound, codes.Unauthenticated,
            codes.Unimplemented:
           return false // Non-retryable
       default:
           return true  // Retryable (Internal, Unavailable, etc.)
       }
   }
   ```

2. **Replace the hardcoded check** in bucket creation and deletion:
   ```go
   // OLD (broken):
   if status.Code(err) != codes.AlreadyExists {
       return b.recordError(...)
   }
   
   // NEW (fixed):
   if !isRetryableError(err) {
       return b.recordError(...)
   }
   // For retryable errors, return error to trigger backoff
   return fmt.Errorf("failed to create bucket (retryable): %w", err)
   ```

#### **Implementation Steps**
1. **Fork the COSI sidecar repository**
2. **Apply the error classification fix**
3. **Build and push a patched sidecar image** (e.g., `ghcr.io/scality/cosi-sidecar:fixed`)
4. **Update our deployment to use the fixed sidecar**

### **Option 2: Workaround** (Not Recommended)
Modify our driver to return `codes.AlreadyExists` for all non-retryable errors, but this would be semantically incorrect.

## 🚀 **Deployment Fix**

### **Current Deployment** (Broken)
```yaml
- name: objectstorage-provisioner-sidecar
  image: gcr.io/k8s-staging-sig-storage/objectstorage-sidecar:v20241219-v0.1.0-60-g6a5a12c
```

### **Fixed Deployment** (After implementing Option 1)
```yaml
- name: objectstorage-provisioner-sidecar
  image: ghcr.io/scality/cosi-sidecar:fixed
```

## 📊 **Impact Assessment**

### **Before Fix**
- ❌ All non-`AlreadyExists` errors cause infinite retry
- ❌ `InvalidBucketName` errors retry forever
- ❌ `PermissionDenied` errors retry forever  
- ❌ `BucketNotEmpty` errors retry forever
- ❌ Controller logs filled with retry attempts
- ❌ Resource exhaustion from exponential backoff

### **After Fix**
- ✅ Proper error classification prevents infinite retries
- ✅ Client errors (InvalidBucketName) fail immediately
- ✅ Permission errors fail immediately
- ✅ State conflicts (BucketNotEmpty) fail immediately
- ✅ Only truly retryable errors (network issues) retry
- ✅ Clean controller logs
- ✅ Proper resource utilization

## 🎯 **Next Steps**

1. **Immediate**: Implement the sidecar fix (Option 1)
2. **Test**: Verify the fix resolves infinite retries
3. **Upstream**: Consider contributing the fix back to kubernetes-sigs/container-object-storage-interface
4. **Documentation**: Update our deployment guides with the fixed sidecar

## 📝 **Files Modified**

### **Our Driver (Already Fixed)** ✅
- `pkg/driver/provisioner_server_impl.go` - Enhanced error classification
- `pkg/driver/provisioner_server_impl_test.go` - Comprehensive test coverage

### **COSI Sidecar (Needs Fix)** ❌
- `sidecar/pkg/bucket/bucket_listener.go` - Add proper error classification
- Update deployment manifests to use fixed sidecar image

## 🏆 **Conclusion**

The COSI-86 infinite retry issue is a **two-part problem**:
1. **Driver-side** (✅ FIXED): Return proper gRPC error codes
2. **Sidecar-side** (❌ NEEDS FIX): Handle proper gRPC error codes

**Our driver fix was correct and necessary, but insufficient. The sidecar fix is required to completely resolve COSI-86.** 