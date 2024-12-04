package util_test

import (
	"github.com/scality/cosi-driver/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("StorageClientParameters", func() {
	Context("NewStorageClientParameters", func() {
		It("should initialize default parameters", func() {
			params := util.NewStorageClientParameters()

			Expect(params.Region).To(Equal(util.DefaultRegion))
			Expect(params.Debug).To(BeFalse())
			Expect(params.AccessKeyID).To(BeEmpty())
			Expect(params.SecretAccessKey).To(BeEmpty())
			Expect(params.Endpoint).To(BeEmpty())
			Expect(params.TLSCert).To(BeNil())
		})
	})

	Context("Validate", func() {
		var params util.StorageClientParameters

		BeforeEach(func() {
			params = *util.NewStorageClientParameters()
		})

		It("should validate successfully when all required fields are set", func() {
			params.AccessKeyID = "test-access-key"
			params.SecretAccessKey = "test-secret-key"
			params.Endpoint = "https://test-endpoint"

			err := params.Validate()
			Expect(err).To(BeNil())
		})

		It("should return error when AccessKeyID is missing", func() {
			params.SecretAccessKey = "test-secret-key"
			params.Endpoint = "https://test-endpoint"

			err := params.Validate()
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
			Expect(err.Error()).To(ContainSubstring("accessKeyID is required"))
		})

		It("should return error when SecretAccessKey is missing", func() {
			params.AccessKeyID = "test-access-key"
			params.Endpoint = "https://test-endpoint"

			err := params.Validate()
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
			Expect(err.Error()).To(ContainSubstring("secretAccessKey is required"))
		})

		It("should return error when Endpoint is missing", func() {
			params.AccessKeyID = "test-access-key"
			params.SecretAccessKey = "test-secret-key"

			err := params.Validate()
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
			Expect(err.Error()).To(ContainSubstring("endpoint is required"))
		})

		It("should allow optional TLSCert", func() {
			params.AccessKeyID = "test-access-key"
			params.SecretAccessKey = "test-secret-key"
			params.Endpoint = "https://test-endpoint"
			params.TLSCert = []byte("mock-cert")

			err := params.Validate()
			Expect(err).To(BeNil())
		})

		It("should treat empty strings as missing fields", func() {
			params.AccessKeyID = ""
			params.SecretAccessKey = "test-secret-key"
			params.Endpoint = "https://test-endpoint"

			err := params.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("accessKeyID is required"))
		})
	})
})
