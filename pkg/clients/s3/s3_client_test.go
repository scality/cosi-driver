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
	s3client "github.com/scality/cosi-driver/pkg/clients/s3"
	"github.com/scality/cosi-driver/pkg/util"
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
	RunSpecs(t, "S3Client Test Suite")
}

var _ = Describe("S3Client", func() {

	var params util.StorageClientParameters

	BeforeEach(func() {
		params = *util.NewStorageClientParameters()
		// Override fields as needed for the test
		params.AccessKeyID = "test-access-key"
		params.SecretAccessKey = "test-secret-key"
		params.Endpoint = "https://s3.mock.endpoint"
		params.TLSCert = nil
		params.Debug = false
	})

	Describe("InitS3Client", func() {
		It("should initialize the S3 client without error", func() {
			client, err := s3client.InitS3Client(params)
			Expect(err).To(BeNil())
			Expect(client).NotTo(BeNil())
			Expect(client).To(BeAssignableToTypeOf(&s3client.S3Client{}))
		})

		It("should use the default region when none is provided", func() {
			client, err := s3client.InitS3Client(params)
			Expect(err).To(BeNil())
			Expect(client).NotTo(BeNil())
			Expect(client.S3Service).NotTo(BeNil())
			opts := client.S3Service.(*s3.Client).Options() // print the opts to see the region
			Expect(opts.Region).To(Equal("us-east-1"))
		})
	})

	Describe("CreateBucket", func() {
		var mockS3 *MockS3Client

		BeforeEach(func() {
			mockS3 = &MockS3Client{}
			params.Region = "us-west-2"
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
