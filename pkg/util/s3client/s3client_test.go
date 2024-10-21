package s3client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scality/cosi/pkg/util/s3client"
)

// MockS3Client implements the S3API interface for testing
type MockS3Client struct {
	CreateBucketFunc func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

func (m *MockS3Client) CreateBucket(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	if m.CreateBucketFunc != nil {
		return m.CreateBucketFunc(ctx, input, opts...)
	}
	return &s3.CreateBucketOutput{}, nil
}

func TestS3Client(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3Client Suite")
}

var _ = Describe("S3Client", func() {

	var params s3client.S3Params

	BeforeEach(func() {
		params = s3client.S3Params{
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
			Endpoint:  "https://s3.mock.endpoint",
			Region:    "us-west-2",
			TLSCert:   nil,
			Debug:     false,
		}
	})

	Describe("InitS3Client", func() {
		It("should initialize the S3 client without error", func() {
			client, err := s3client.InitS3Client(params)
			Expect(err).To(BeNil())
			Expect(client).NotTo(BeNil())
			Expect(client).To(BeAssignableToTypeOf(&s3client.S3Client{}))
		})

		It("should use the default region when none is provided", func() {
			params.Region = ""
			client, err := s3client.InitS3Client(params)
			Expect(err).To(BeNil())
			Expect(client).NotTo(BeNil())
			Expect(client.S3Service).NotTo(BeNil())
			opts := client.S3Service.(*s3.Client).Options()
			Expect(opts.Region).To(Equal("us-east-1"))
		})

		It("should fail if credentials are missing", func() {
			params.AccessKey = ""
			params.SecretKey = ""
			client, err := s3client.InitS3Client(params)
			Expect(err).NotTo(BeNil())
			Expect(client).To(BeNil())
		})
	})

	Describe("ConfigureTLSTransport", func() {

		It("should configure TLS when certData is provided", func() {
			// Fake certificate data
			certData := []byte("fake-cert-data")
			transport := s3client.ConfigureTLSTransport(certData, false)

			Expect(transport).NotTo(BeNil())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			Expect(transport.TLSClientConfig.RootCAs).NotTo(BeNil())
		})

		It("should skip TLS validation when no certData is provided and skipTLSValidation is true", func() {
			transport := s3client.ConfigureTLSTransport(nil, true)

			Expect(transport).NotTo(BeNil())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
			Expect(transport.TLSClientConfig.RootCAs).To(BeNil())
		})

		It("should not configure TLS when no certData is provided and skipTLSValidation is false", func() {
			transport := s3client.ConfigureTLSTransport(nil, false)

			Expect(transport).NotTo(BeNil())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			Expect(transport.TLSClientConfig.RootCAs).To(BeNil())
		})

	})

	Describe("CreateBucket", func() {
		var mockS3 *MockS3Client

		BeforeEach(func() {
			mockS3 = &MockS3Client{}
			client, _ := s3client.InitS3Client(params)
			client.S3Service = mockS3
		})

		It("should successfully create a bucket in a non-us-east-1 region", func(ctx SpecContext) {
			mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
				Expect(input.Bucket).To(Equal(aws.String("new-bucket")))
				Expect(input.CreateBucketConfiguration.LocationConstraint).To(Equal(types.BucketLocationConstraint("us-west-2")))
				return &s3.CreateBucketOutput{}, nil
			}

			client, _ := s3client.InitS3Client(params)
			client.S3Service = mockS3

			err := client.CreateBucket(ctx, "new-bucket", params)
			Expect(err).To(BeNil())
		})

		It("should handle other errors correctly", func(ctx SpecContext) {
			mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
				return nil, fmt.Errorf("SomeOtherError: Something went wrong")
			}

			client, _ := s3client.InitS3Client(params)
			client.S3Service = mockS3

			err := client.CreateBucket(ctx, "new-bucket", params)
			Expect(err).NotTo(BeNil())
		})
	})
})
