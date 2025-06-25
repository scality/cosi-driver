package osperrors

import (
	"google.golang.org/grpc/codes"
)

// IAMErrorTable maps AWS IAM error codes to gRPC codes and message templates.
// Error codes match those returned by AWS SDK's smithy.APIError.ErrorCode().
// Code mappings follow https://pkg.go.dev/google.golang.org/grpc@v1.72.1/codes
var IAMErrorTable = map[string]ObjectStorageProviderError{
	// AlreadyExists: entity already exists
	"EntityAlreadyExists": {
		GRPCCode:         codes.AlreadyExists,
		LogMessage:       "IAM entity already exists",
		ClientMessageTpl: "IAM user %s already exists",
	},
	"AccessKeyAlreadyExists": {
		GRPCCode:         codes.AlreadyExists,
		LogMessage:       "Access key already exists",
		ClientMessageTpl: "access key already exists for user %s",
	},

	// InvalidArgument: client specified invalid argument
	"InvalidInput": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Invalid input parameters",
		ClientMessageTpl: "invalid input for IAM user %s",
	},
	"InvalidUserName": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Invalid user name format",
		ClientMessageTpl: "invalid user name: %s",
	},
	"ValidationError": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Request validation failed",
		ClientMessageTpl: "validation error for user %s",
	},
	"MalformedPolicyDocument": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Policy document is malformed",
		ClientMessageTpl: "malformed policy document for user %s",
	},
	"PolicyNotAttachable": {
		GRPCCode:         codes.InvalidArgument,
		LogMessage:       "Policy cannot be attached",
		ClientMessageTpl: "policy not attachable for user %s",
	},

	// ResourceExhausted: resource quota exceeded
	"LimitExceeded": {
		GRPCCode:         codes.ResourceExhausted,
		LogMessage:       "IAM service limit exceeded",
		ClientMessageTpl: "IAM limit exceeded for user %s",
	},
	"AccessKeyLimitExceeded": {
		GRPCCode:         codes.ResourceExhausted,
		LogMessage:       "Access key limit per user exceeded",
		ClientMessageTpl: "access key limit exceeded for user %s",
	},
	"Throttled": {
		GRPCCode:         codes.ResourceExhausted,
		LogMessage:       "IAM request throttled - rate limit exceeded",
		ClientMessageTpl: "IAM request throttled for user %s",
	},

	// PermissionDenied: caller lacks permission
	"AccessDenied": {
		GRPCCode:         codes.PermissionDenied,
		LogMessage:       "Access denied for IAM operation",
		ClientMessageTpl: "permission denied for IAM operation on user %s",
	},
	"UnauthorizedOperation": {
		GRPCCode:         codes.PermissionDenied,
		LogMessage:       "Operation not authorized",
		ClientMessageTpl: "unauthorized IAM operation for user %s",
	},

	// FailedPrecondition: system not in required state
	"DeleteConflict": {
		GRPCCode:         codes.FailedPrecondition,
		LogMessage:       "Cannot delete user with attached resources",
		ClientMessageTpl: "user %s has attached resources",
	},

	// Unavailable: service temporarily unavailable
	"ServiceUnavailable": {
		GRPCCode:         codes.Unavailable,
		LogMessage:       "IAM service temporarily unavailable",
		ClientMessageTpl: "IAM service temporarily unavailable for user %s",
	},
	"EntityTemporarilyUnmodifiable": {
		GRPCCode:         codes.Unavailable,
		LogMessage:       "Entity temporarily unmodifiable - retry later",
		ClientMessageTpl: "user %s is temporarily unmodifiable",
	},

	// Internal: internal service error
	"ServiceFailure": {
		GRPCCode:         codes.Internal,
		LogMessage:       "IAM service internal failure",
		ClientMessageTpl: "IAM service failure for user %s",
	},

	// OK: treat as success for idempotency
	"NoSuchEntity": {
		GRPCCode:         codes.OK,
		LogMessage:       "IAM entity does not exist - idempotent success",
		ClientMessageTpl: "",
	},
}

// TranslateIAMError translates AWS IAM errors to gRPC status errors
func TranslateIAMError(action, userName string, err error) error {
	return TranslateObjectStorageProviderError(action, userName, "IAM", err, IAMErrorTable)
}
