# Upstream Contribution Plan - COSI Error Handling Fix

## 🎯 Goal
Fix the COSI sidecar's overly broad retry logic that causes infinite retry loops for permanent errors like `InvalidArgument`, `PermissionDenied`, etc.

## 📋 Contribution Strategy

### Phase 1: Issue Creation
1. **Create GitHub Issue** in `kubernetes-sigs/container-object-storage-interface`
   - Title: `[Bug] Sidecar retries all non-AlreadyExists errors infinitely`
   - Label: `kind/bug`, `area/sidecar`
   - Include reproduction case with `InvalidBucketName`
   - Link to this COSI-86 issue as real-world impact

### Phase 2: RFC/Discussion
1. **Create RFC Issue** for the fix
   - Title: `[RFC] Improve sidecar error handling to only retry Internal errors`
   - Reference existing issue
   - Propose the conservative "only Internal errors retry" approach
   - Get community consensus before implementation

### Phase 3: Implementation
1. **Fork Repository**
   ```bash
   git clone https://github.com/kubernetes-sigs/container-object-storage-interface.git
   cd container-object-storage-interface
   git checkout -b fix/sidecar-error-handling
   ```

2. **Implement Fix**
   - Add `isRetryableError()` function
   - Update bucket creation/deletion error handling
   - Add comprehensive test cases
   - Update documentation

3. **Testing Strategy**
   - Unit tests for `isRetryableError()` function
   - Integration tests for each error code
   - E2E tests with mock driver returning various error codes
   - Ensure backward compatibility

### Phase 4: Pull Request
1. **PR Requirements**
   - Clear title: `Fix sidecar infinite retry for permanent errors`
   - Detailed description linking to RFC
   - Test coverage report
   - Documentation updates
   - Changelog entry

## 📝 Issue Template

```markdown
# Sidecar retries all non-AlreadyExists errors infinitely

## Problem
The COSI sidecar has overly broad retry logic that only treats `codes.AlreadyExists` as non-retryable. This causes infinite retry loops for legitimate permanent errors.

## Current Behavior
```go
if status.Code(err) != codes.AlreadyExists {
    return b.recordError(inputBucket, v1.EventTypeWarning, v1alpha1.FailedCreateBucket, fmt.Errorf("failed to create bucket: %w", err))
}
```

**All errors except `AlreadyExists` trigger infinite retries**, including:
- `InvalidArgument` (bad bucket names, invalid parameters)
- `PermissionDenied` (access denied) 
- `FailedPrecondition` (bucket not empty)
- `ResourceExhausted` (quota exceeded)

## Impact
- **Resource Exhaustion**: Exponential backoff consumes cluster resources
- **Poor UX**: No clear failure indication to users
- **Operational Issues**: Infinite retry loops mask real problems

## Real-World Example
Scality COSI driver (COSI-86): Invalid bucket names cause infinite retries despite driver correctly returning `codes.InvalidArgument`.

## Proposed Solution
**Conservative Approach**: Only `codes.Internal` errors should trigger retries.

**Rationale**: Only `codes.Internal` represents transient failures (network issues, temporary backend problems). All other gRPC codes represent permanent failures that won't be resolved by retrying.

## Error Code Classification
| Code | Should Retry? | Reasoning |
|------|---------------|-----------|
| `Internal` | ✅ YES | Transient failures, network issues |
| `InvalidArgument` | ❌ NO | Bad parameters won't fix themselves |
| `PermissionDenied` | ❌ NO | Access denied won't change |
| `FailedPrecondition` | ❌ NO | State conflicts require intervention |
| `ResourceExhausted` | ❌ NO | Quota limits require admin action |
| `AlreadyExists` | ❌ NO | Success for idempotent operations |
| All others | ❌ NO | Conservative approach |
```

## 🧪 Test Cases

```go
func TestIsRetryableError(t *testing.T) {
    tests := []struct {
        name      string
        code      codes.Code
        retryable bool
    }{
        {"Internal should retry", codes.Internal, true},
        {"InvalidArgument should not retry", codes.InvalidArgument, false},
        {"PermissionDenied should not retry", codes.PermissionDenied, false},
        {"FailedPrecondition should not retry", codes.FailedPrecondition, false},
        {"ResourceExhausted should not retry", codes.ResourceExhausted, false},
        {"AlreadyExists should not retry", codes.AlreadyExists, false},
        {"NotFound should not retry", codes.NotFound, false},
        {"Unavailable should not retry", codes.Unavailable, false},
        {"Unknown should not retry", codes.Unknown, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := status.Error(tt.code, "test error")
            if got := isRetryableError(err); got != tt.retryable {
                t.Errorf("isRetryableError() = %v, want %v", got, tt.retryable)
            }
        })
    }
}
```

## ⏱️ Timeline
- **Week 1**: Create issue and RFC, gather feedback
- **Week 2**: Implement fix and tests
- **Week 3**: Create PR and address review feedback  
- **Week 4**: Final review and merge

## 🔄 Immediate Workaround
While waiting for upstream fix, teams can:
1. Build custom sidecar with the patch
2. Use the fixed image in their deployments
3. Switch back to upstream once fix is merged

## Benefits
- ✅ Fixes infinite retry loops
- ✅ Improves resource utilization
- ✅ Better error visibility
- ✅ Backward compatible
- ✅ Benefits entire COSI ecosystem 