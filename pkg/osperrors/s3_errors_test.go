package osperrors_test

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	constants "github.com/scality/cosi-driver/pkg/constants"
	"github.com/scality/cosi-driver/pkg/osperrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("S3 Error Translation", func() {
	Describe("TranslateS3Error", func() {
		Context("when handling bucket creation errors", func() {
			It("should translate BucketAlreadyExists error", func() {
				err := &types.BucketAlreadyExists{}
				expected := status.Errorf(codes.AlreadyExists, "bucket %s already exists", "test-bucket")

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.AlreadyExists))
				Expect(result.Error()).To(Equal(expected.Error()))
			})

			It("should translate InvalidBucketName error", func() {
				err := &mockAPIError{code: "InvalidBucketName", message: "Invalid bucket name"}
				expected := status.Errorf(codes.InvalidArgument, "invalid bucket name: %s", "test-bucket")

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(Equal(expected.Error()))
			})

			It("should translate InvalidLocationConstraint error", func() {
				err := &mockAPIError{code: "InvalidLocationConstraint", message: "Invalid location"}

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(ContainSubstring("invalid location constraint"))
			})

			It("should translate AccessDenied error", func() {
				err := &mockAPIError{code: "AccessDenied", message: "Access denied"}

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.PermissionDenied))
				Expect(result.Error()).To(ContainSubstring("permission denied"))
			})

			It("should translate InvalidRequest error", func() {
				err := &mockAPIError{code: "InvalidRequest", message: "Invalid request"}

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(ContainSubstring("invalid request"))
			})

			It("should translate MalformedXML error", func() {
				err := &mockAPIError{code: "MalformedXML", message: "Malformed XML"}

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.InvalidArgument))
				Expect(result.Error()).To(ContainSubstring("malformed XML"))
			})
		})

		Context("when handling bucket deletion errors", func() {
			It("should translate BucketNotEmpty error", func() {
				err := &mockAPIError{code: "BucketNotEmpty", message: "Bucket not empty"}

				result := osperrors.TranslateS3Error(constants.ActionDeleteBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.FailedPrecondition))
				Expect(result.Error()).To(ContainSubstring("bucket test-bucket is not empty"))
			})

			It("should treat NoSuchBucket as success", func() {
				err := &mockAPIError{code: "NoSuchBucket", message: "No such bucket"}

				result := osperrors.TranslateS3Error(constants.ActionDeleteBucket, "test-bucket", err)

				Expect(result).To(BeNil())
			})

			It("should treat NotFound as success", func() {
				err := &mockAPIError{code: "NotFound", message: "Not found"}

				result := osperrors.TranslateS3Error(constants.ActionDeleteBucket, "test-bucket", err)

				Expect(result).To(BeNil())
			})
		})

		Context("when handling unrecognized errors", func() {
			It("should return internal error for unknown S3 error", func() {
				err := &mockAPIError{code: "UnknownError", message: "Unknown error"}

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.Internal))
				Expect(result.Error()).To(ContainSubstring("failed to CreateBucket resource"))
			})

			It("should handle non-API errors", func() {
				err := errors.New("random error")

				result := osperrors.TranslateS3Error(constants.ActionCreateBucket, "test-bucket", err)

				Expect(result).NotTo(BeNil())
				Expect(status.Code(result)).To(Equal(codes.Internal))
				Expect(result.Error()).To(ContainSubstring("unexpected error"))
			})
		})
	})

	Describe("S3ErrorTable", func() {
		It("should contain all expected error codes", func() {
			expectedErrorCodes := []string{
				"BucketAlreadyExists",
				"BucketAlreadyOwnedByYou",
				"InvalidBucketName",
				"InvalidLocationConstraint",
				"AccessDenied",
				"InvalidRequest",
				"MalformedXML",
				"BucketNotEmpty",
				"NoSuchBucket",
				"NotFound",
				"RequestTimeout",
				"ServiceUnavailable",
				"Throttled",
				"TooManyBuckets",
				"InvalidAccessKeyId",
				"SignatureDoesNotMatch",
			}

			for _, code := range expectedErrorCodes {
				_, exists := osperrors.S3ErrorTable[code]
				Expect(exists).To(BeTrue(), "Expected error code %s to be in S3ErrorTable", code)
			}
		})
	})
})
