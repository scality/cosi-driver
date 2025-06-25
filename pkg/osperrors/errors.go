// Package osperrors provides error translation for Object Storage Provider (OSP) errors.
// OSP refers to S3-compatible storage systems and IAM services that implement AWS-style APIs.
// This package maps OSP-specific error codes to appropriate gRPC status codes to ensure
// proper error handling and prevent infinite retry loops in the COSI controller.
package osperrors

import (
	"errors"
	"fmt"

	smithy "github.com/aws/smithy-go"
	constants "github.com/scality/cosi-driver/pkg/constants"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// ObjectStorageProviderError represents metadata for object storage provider error mapping
type ObjectStorageProviderError struct {
	GRPCCode         codes.Code
	LogMessage       string
	ClientMessageTpl string // Template for client-facing error messages with %s placeholder for resource name
}

// Common error messages to reduce allocations
const (
	unexpectedErrorMsg = "unexpected error"
	failedToMsgFmt     = "failed to %s resource"
)

// TranslateObjectStorageProviderError translates object storage provider errors to gRPC status errors.
// It expects errors implementing smithy.APIError interface (AWS SDK errors).
// Returns nil for errors mapped to codes.OK (idempotent operations).
func TranslateObjectStorageProviderError(action, resourceName, provider string, err error, errorTable map[string]ObjectStorageProviderError) error {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		klog.ErrorS(err, "Unhandled error", "action", action, "resourceName", resourceName, "provider", provider)
		return status.Error(codes.Internal, unexpectedErrorMsg)
	}

	errorCode := apiErr.ErrorCode()
	meta, ok := errorTable[errorCode]
	if !ok {
		// Log with structured fields for better observability
		klog.ErrorS(err, "Unrecognized error code",
			"action", action,
			"resourceName", resourceName,
			"provider", provider,
			"errorCode", errorCode)
		return status.Error(codes.Internal, fmt.Sprintf(failedToMsgFmt, action))
	}

	// Special handling for operation-specific idempotent errors
	// NoSuchBucket, NotFound, and NoSuchEntity should only be treated as idempotent success for delete/revoke operations
	isIdempotentAction := action == constants.ActionDeleteBucket || action == constants.ActionRevokeBucketAccess || action == "delete"
	if (errorCode == "NoSuchBucket" || errorCode == "NotFound" || errorCode == "NoSuchEntity") && !isIdempotentAction {
		// For non-delete/revoke operations, treat these as NotFound errors instead of idempotent success
		klog.ErrorS(err, "Resource not found during "+action+" operation",
			"resourceName", resourceName,
			"action", action,
			"errorCode", errorCode)
		return status.Errorf(codes.NotFound, "resource %s not found", resourceName)
	}

	// Handle idempotent operations
	if meta.GRPCCode == codes.OK {
		klog.V(constants.LvlInfo).InfoS(meta.LogMessage, "resourceName", resourceName, "action", action)
		return nil
	}

	// Log error with structured fields
	klog.ErrorS(err, meta.LogMessage,
		"resourceName", resourceName,
		"action", action,
		"errorCode", errorCode)

	if meta.ClientMessageTpl != "" {
		return status.Errorf(meta.GRPCCode, meta.ClientMessageTpl, resourceName)
	}
	return status.Error(meta.GRPCCode, meta.LogMessage)
}
