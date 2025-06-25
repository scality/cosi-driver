package osperrors_test

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scality/cosi-driver/pkg/osperrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("IAM Error Translation", func() {
	Describe("TranslateIAMError", func() {
		Context("when handling user creation errors", func() {
			It("should translate EntityAlreadyExists error", func() {
				err := &types.EntityAlreadyExistsException{}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.AlreadyExists))
				Expect(result.Error()).To(ContainSubstring("IAM user test-user already exists"))
			})

			It("should translate InvalidInput error", func() {
				err := &mockAPIError{code: "InvalidInput", message: "Invalid input"}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(ContainSubstring("invalid input"))
			})

			It("should translate LimitExceeded error", func() {
				err := &types.LimitExceededException{}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.ResourceExhausted))
				Expect(result.Error()).To(ContainSubstring("IAM limit exceeded"))
			})

			It("should translate ServiceFailure error", func() {
				err := &types.ServiceFailureException{}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.Internal))
				Expect(result.Error()).To(ContainSubstring("IAM service failure"))
			})
		})

		Context("when handling user deletion errors", func() {
			It("should treat NoSuchEntity as success", func() {
				err := &types.NoSuchEntityException{}

				result := osperrors.TranslateIAMError("delete", "test-user", err)

				Expect(result).To(BeNil())
			})

			It("should translate DeleteConflict error", func() {
				err := &types.DeleteConflictException{}

				result := osperrors.TranslateIAMError("delete", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.FailedPrecondition))
				Expect(result.Error()).To(ContainSubstring("user test-user has attached resources"))
			})
		})

		Context("when handling access key errors", func() {
			It("should translate AccessKeyAlreadyExists error", func() {
				err := &mockAPIError{code: "AccessKeyAlreadyExists", message: "Access key already exists"}

				result := osperrors.TranslateIAMError("create access key", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.AlreadyExists))
				Expect(result.Error()).To(ContainSubstring("access key already exists"))
			})

			It("should translate AccessKeyLimitExceeded error", func() {
				err := &mockAPIError{code: "AccessKeyLimitExceeded", message: "Access key limit exceeded"}

				result := osperrors.TranslateIAMError("create access key", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.ResourceExhausted))
				Expect(result.Error()).To(ContainSubstring("access key limit exceeded"))
			})
		})

		Context("when handling policy errors", func() {
			It("should translate MalformedPolicyDocument error", func() {
				err := &types.MalformedPolicyDocumentException{}

				result := osperrors.TranslateIAMError("attach policy", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(ContainSubstring("malformed policy document"))
			})

			It("should translate PolicyNotAttachable error", func() {
				err := &mockAPIError{code: "PolicyNotAttachable", message: "Policy not attachable"}

				result := osperrors.TranslateIAMError("attach policy", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(ContainSubstring("policy not attachable"))
			})
		})

		Context("when handling permission errors", func() {
			It("should translate AccessDenied error", func() {
				err := &mockAPIError{code: "AccessDenied", message: "Access denied"}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.PermissionDenied))
				Expect(result.Error()).To(ContainSubstring("permission denied"))
			})

			It("should translate UnauthorizedOperation error", func() {
				err := &mockAPIError{code: "UnauthorizedOperation", message: "Unauthorized"}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.PermissionDenied))
				Expect(result.Error()).To(ContainSubstring("unauthorized"))
			})
		})

		Context("when handling unrecognized errors", func() {
			It("should return internal error for unknown IAM error", func() {
				err := &mockAPIError{code: "UnknownError", message: "Unknown error"}

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.Internal))
				Expect(result.Error()).To(ContainSubstring("failed to create resource"))
			})

			It("should handle non-API errors", func() {
				err := errors.New("random error")

				result := osperrors.TranslateIAMError("create", "test-user", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.Internal))
				Expect(result.Error()).To(ContainSubstring("unexpected error"))
			})
		})
	})

	Describe("IAMErrorTable", func() {
		It("should contain all expected error codes", func() {
			expectedErrorCodes := []string{
				"EntityAlreadyExists",
				"InvalidInput",
				"LimitExceeded",
				"ServiceFailure",
				"NoSuchEntity",
				"DeleteConflict",
				"AccessKeyAlreadyExists",
				"AccessKeyLimitExceeded",
				"MalformedPolicyDocument",
				"PolicyNotAttachable",
				"AccessDenied",
				"UnauthorizedOperation",
				"InvalidUserName",
				"ValidationError",
				"EntityTemporarilyUnmodifiable",
				"ServiceUnavailable",
				"Throttled",
			}

			for _, code := range expectedErrorCodes {
				_, exists := osperrors.IAMErrorTable[code]
				Expect(exists).To(BeTrue(), "Expected error code %s to be in IAMErrorTable", code)
			}
		})
	})
})
