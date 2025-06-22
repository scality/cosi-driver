package osperrors_test

import (
	"errors"

	smithy "github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	constants "github.com/scality/cosi-driver/pkg/constants"
	"github.com/scality/cosi-driver/pkg/osperrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mock smithy.APIError implementation
type mockAPIError struct {
	code    string
	message string
	fault   smithy.ErrorFault
}

func (e *mockAPIError) Error() string {
	return e.message
}

func (e *mockAPIError) ErrorCode() string {
	return e.code
}

func (e *mockAPIError) ErrorMessage() string {
	return e.message
}

func (e *mockAPIError) ErrorFault() smithy.ErrorFault {
	return e.fault
}

var _ = Describe("TranslateObjectStorageProviderError", func() {
	var testErrorTable map[string]osperrors.ObjectStorageProviderError

	BeforeEach(func() {
		testErrorTable = map[string]osperrors.ObjectStorageProviderError{
			"TestAlreadyExists": {
				GRPCCode:         codes.AlreadyExists,
				LogMessage:       "Resource already exists",
				ClientMessageTpl: "Resource already exists: %s",
			},
			"TestNotFound": {
				GRPCCode:         codes.OK, // Treat as success
				LogMessage:       "Resource not found - treated as success",
				ClientMessageTpl: "",
			},
			"TestAccessDenied": {
				GRPCCode:         codes.PermissionDenied,
				LogMessage:       "Access denied",
				ClientMessageTpl: "Access denied for resource: %s",
			},
			"TestNoTemplate": {
				GRPCCode:         codes.Internal,
				LogMessage:       "Internal error without template",
				ClientMessageTpl: "",
			},
		}
	})

	Context("when handling recognized errors", func() {
		It("should translate error with template correctly", func() {
			err := &mockAPIError{code: "TestAlreadyExists", message: "Already exists"}
			expected := status.Errorf(codes.AlreadyExists, "Resource already exists: %s", "test-resource")

			result := osperrors.TranslateObjectStorageProviderError("create", "test-resource", "TestProvider", err, testErrorTable)

			Expect(result).NotTo(BeNil())
			Expect(status.Code(result)).To(Equal(codes.AlreadyExists))
			Expect(result.Error()).To(Equal(expected.Error()))
		})

		It("should treat certain errors as success", func() {
			err := &mockAPIError{code: "TestNotFound", message: "Not found"}

			result := osperrors.TranslateObjectStorageProviderError("delete", "test-resource", "TestProvider", err, testErrorTable)

			Expect(result).To(BeNil())
		})

		It("should handle error without template", func() {
			err := &mockAPIError{code: "TestNoTemplate", message: "Internal error"}
			expected := status.Error(codes.Internal, "Internal error without template")

			result := osperrors.TranslateObjectStorageProviderError("update", "test-resource", "TestProvider", err, testErrorTable)

			Expect(result).NotTo(BeNil())
			Expect(status.Code(result)).To(Equal(codes.Internal))
			Expect(result.Error()).To(Equal(expected.Error()))
		})

		It("should map NoSuchBucket to NotFound for non-delete operations", func() {
			apiErr := &smithy.GenericAPIError{Code: "NoSuchBucket", Message: "The specified bucket does not exist"}

			err := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", apiErr)
			Expect(err).NotTo(BeNil())

			statusErr, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(statusErr.Code()).To(Equal(codes.NotFound))
			Expect(statusErr.Message()).To(Equal("resource test-bucket not found"))
		})

		It("should map NoSuchBucket to OK for delete operations (idempotent success)", func() {
			apiErr := &smithy.GenericAPIError{Code: "NoSuchBucket", Message: "The specified bucket does not exist"}

			err := osperrors.TranslateS3Error(constants.ActionDeleteBucket, "test-bucket", apiErr)
			Expect(err).To(BeNil())
		})

		It("should map NotFound to NotFound for non-delete operations", func() {
			apiErr := &smithy.GenericAPIError{Code: "NotFound", Message: "The resource was not found"}

			err := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", apiErr)
			Expect(err).NotTo(BeNil())

			statusErr, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(statusErr.Code()).To(Equal(codes.NotFound))
			Expect(statusErr.Message()).To(Equal("resource test-bucket not found"))
		})

		It("should map NoSuchEntity to NotFound for non-revoke operations", func() {
			apiErr := &smithy.GenericAPIError{Code: "NoSuchEntity", Message: "The user does not exist"}

			err := osperrors.TranslateIAMError(constants.ActionGrantBucketAccess, "test-user", apiErr)
			Expect(err).NotTo(BeNil())

			statusErr, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(statusErr.Code()).To(Equal(codes.NotFound))
			Expect(statusErr.Message()).To(Equal("resource test-user not found"))
		})

		It("should map NoSuchEntity to OK for revoke access operations (idempotent success)", func() {
			apiErr := &smithy.GenericAPIError{Code: "NoSuchEntity", Message: "The user does not exist"}

			err := osperrors.TranslateIAMError(constants.ActionRevokeBucketAccess, "test-user", apiErr)
			Expect(err).To(BeNil())
		})
	})

	Context("when handling unrecognized errors", func() {
		It("should return internal error for unknown API error", func() {
			err := &mockAPIError{code: "UnknownError", message: "Unknown error"}

			result := osperrors.TranslateObjectStorageProviderError("create", "test-resource", "TestProvider", err, testErrorTable)

			Expect(result).NotTo(BeNil())
			Expect(status.Code(result)).To(Equal(codes.Internal))
			Expect(result.Error()).To(ContainSubstring("failed to create resource"))
		})

		It("should handle non-API errors", func() {
			err := errors.New("random error")

			result := osperrors.TranslateObjectStorageProviderError("create", "test-resource", "TestProvider", err, testErrorTable)

			Expect(result).NotTo(BeNil())
			Expect(status.Code(result)).To(Equal(codes.Internal))
			Expect(result.Error()).To(ContainSubstring("unexpected error"))
		})
	})
})
