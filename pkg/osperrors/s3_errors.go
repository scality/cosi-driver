package osperrors

import (
	"google.golang.org/grpc/codes"
)

// S3ErrorTable maps AWS S3 error codes to gRPC codes and message templates.
// Error codes match those returned by AWS SDK's smithy.APIError.ErrorCode().
// Code mappings follow https://pkg.go.dev/google.golang.org/grpc@v1.72.1/codes
var S3ErrorTable = map[string]ObjectStorageProviderError{
	// AlreadyExists: entity already exists
	"BucketAlreadyExists": {
		GRPCCode:         codes.AlreadyExists,
		LogMessage:       "Bucket already exists",
		ClientMessageTpl: "bucket %s already exists",
	},
	"BucketAlreadyOwnedByYou": {
		GRPCCode:         codes.AlreadyExists,
		LogMessage:       "Bucket already owned by you",
		ClientMessageTpl: "bucket %s already owned by you",
	},

	// InvalidArgument: client specified invalid argument
	"InvalidBucketName": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Invalid bucket name",
		ClientMessageTpl: "invalid bucket name: %s",
	},
	"InvalidLocationConstraint": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Invalid location constraint",
		ClientMessageTpl: "invalid location constraint for bucket %s",
	},
	"InvalidRequest": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Invalid request",
		ClientMessageTpl: "invalid request for bucket %s",
	},
	"MalformedXML": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Malformed XML in request",
		ClientMessageTpl: "malformed XML in request for bucket %s",
	},

	// PermissionDenied: caller lacks permission
	"AccessDenied": {
		GRPCCode:         codes.PermissionDenied,
		LogMessage:       "Access denied",
		ClientMessageTpl: "permission denied for bucket %s",
	},

	// FailedPrecondition: system not in required state
	"BucketNotEmpty": {
		GRPCCode:         codes.FailedPrecondition,
		LogMessage:       "Cannot delete non-empty bucket",
		ClientMessageTpl: "bucket %s is not empty",
	},

	// OK: treat as success for idempotency
	"NoSuchBucket": {
		GRPCCode:         codes.OK,
		LogMessage:       "Bucket does not exist - idempotent success",
		ClientMessageTpl: "",
	},
	"NotFound": {
		GRPCCode:         codes.OK,
		LogMessage:       "Resource not found - idempotent success",
		ClientMessageTpl: "",
	},

	// DeadlineExceeded: operation expired
	"RequestTimeout": {
		GRPCCode:         codes.DeadlineExceeded,
		LogMessage:       "Request timeout",
		ClientMessageTpl: "operation timed out for bucket %s",
	},

	// Unavailable: service temporarily unavailable
	"ServiceUnavailable": {
		GRPCCode:         codes.Unavailable,
		LogMessage:       "Service temporarily unavailable",
		ClientMessageTpl: "service temporarily unavailable for bucket %s",
	},

	// ResourceExhausted: resource quota exceeded
	"Throttled": {
		GRPCCode:         codes.ResourceExhausted,
		LogMessage:       "Request throttled - rate limit exceeded",
		ClientMessageTpl: "request throttled for bucket %s",
	},
	"TooManyBuckets": {
		GRPCCode:         codes.ResourceExhausted,
		LogMessage:       "Bucket limit exceeded",
		ClientMessageTpl: "bucket limit exceeded for %s",
	},

	// Unauthenticated: invalid authentication credentials
	"InvalidAccessKeyId": {
		GRPCCode:         codes.Unauthenticated,
		LogMessage:       "Invalid access key",
		ClientMessageTpl: "invalid authentication credentials for bucket %s",
	},
	"SignatureDoesNotMatch": {
		GRPCCode:         codes.Unauthenticated,
		LogMessage:       "Request signature does not match",
		ClientMessageTpl: "authentication signature mismatch for bucket %s",
	},
}

// TranslateS3Error translates AWS S3 errors to gRPC status errors
func TranslateS3Error(action, bucketName string, err error) error {
	return TranslateObjectStorageProviderError(action, bucketName, "S3", err, S3ErrorTable)
}
