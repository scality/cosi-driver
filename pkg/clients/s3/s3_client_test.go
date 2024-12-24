package s3client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	s3client "github.com/scality/cosi-driver/pkg/clients/s3"
	"github.com/scality/cosi-driver/pkg/metrics"
	"github.com/scality/cosi-driver/pkg/mock"
	"github.com/scality/cosi-driver/pkg/util"
)

func TestS3Client(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3Client Test Suite")
}

var _ = BeforeSuite(func() {
	// Initialize metrics globally before all tests
	metrics.InitializeMetrics("test_driver_prefix")
})

var _ = Describe("S3Client", func() {
	var params util.StorageClientParameters

	BeforeEach(func() {
		params = *util.NewStorageClientParameters()
		params.AccessKeyID = "test-access-key"
		params.SecretAccessKey = "test-secret-key"
		params.Endpoint = "https://s3.mock.endpoint"
		params.TLSCert = nil
		params.Debug = false
	})

	Describe("InitS3Client", func() {
		It("should initialize the S3 client without error", func(ctx SpecContext) {
			client, err := s3client.InitS3Client(ctx, params)
			Expect(err).To(BeNil())
			Expect(client).NotTo(BeNil())
			Expect(client).To(BeAssignableToTypeOf(&s3client.S3Client{}))
		})

		It("should use the default region when none is provided", func(ctx SpecContext) {
			client, err := s3client.InitS3Client(ctx, params)
			Expect(err).To(BeNil())
			Expect(client).NotTo(BeNil())
			Expect(client.S3Service).NotTo(BeNil())
			opts := client.S3Service.(*s3.Client).Options()
			Expect(opts.Region).To(Equal("us-east-1"))
		})

		It("should return an error if AWS config loading fails", func(ctx SpecContext) {
			originalLoadAWSConfig := s3client.LoadAWSConfig
			defer func() { s3client.LoadAWSConfig = originalLoadAWSConfig }()

			s3client.LoadAWSConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
				return aws.Config{}, fmt.Errorf("mock config loading error")
			}

			client, err := s3client.InitS3Client(ctx, params)
			Expect(err).To(HaveOccurred())
			Expect(client).To(BeNil())
		})

		It("should set up a logger when Debug is enabled", func(ctx SpecContext) {
			params.Debug = true

			// Mock LoadAWSConfig
			originalLoadAWSConfig := s3client.LoadAWSConfig
			defer func() { s3client.LoadAWSConfig = originalLoadAWSConfig }()

			var loggerUsed bool
			s3client.LoadAWSConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
				// Check if a logger is passed
				for _, optFn := range optFns {
					opt := &config.LoadOptions{}
					optFn(opt)
					if opt.Logger != nil {
						loggerUsed = true
					}
				}
				return aws.Config{}, nil // Simulate a successful load
			}

			_, err := s3client.InitS3Client(ctx, params)
			Expect(err).To(BeNil())
			Expect(loggerUsed).To(BeTrue(), "Expected logger to be used when Debug is enabled")
		})
	})

	Describe("CreateBucket", func() {
		var mockS3 *mock.MockS3Client

		BeforeEach(func(ctx SpecContext) {
			mockS3 = &mock.MockS3Client{}
			params.Region = "us-west-2"
			client, _ := s3client.InitS3Client(ctx, params)
			client.S3Service = mockS3
		})

		It("should successfully create a bucket in a non-us-east-1 region", func(ctx SpecContext) {
			mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
				Expect(input.Bucket).To(Equal(aws.String("new-bucket")))
				Expect(input.CreateBucketConfiguration.LocationConstraint).To(Equal(types.BucketLocationConstraint("us-west-2")))
				return &s3.CreateBucketOutput{}, nil
			}

			client, _ := s3client.InitS3Client(ctx, params)
			client.S3Service = mockS3

			err := client.CreateBucket(ctx, "new-bucket", params)
			Expect(err).To(BeNil())
			// metric := &prometheus.Counter{}
			// Expect(metrics.S3RequestsTotal.WithLabelValues("CreateBucket", "error").Write(metric)).To(Succeed())
			// Expect(metric.GetCounter().GetValue()).To(BeNumerically(">", 0))
		})

		It("should handle other errors correctly", func(ctx SpecContext) {
			mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
				return nil, fmt.Errorf("SomeOtherError: Something went wrong")
			}

			client, _ := s3client.InitS3Client(ctx, params)
			client.S3Service = mockS3

			err := client.CreateBucket(ctx, "new-bucket", params)
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("DeleteBucket", func() {
		var mockS3 *mock.MockS3Client
		var client *s3client.S3Client

		BeforeEach(func() {
			mockS3 = &mock.MockS3Client{}
			client = &s3client.S3Client{
				S3Service: mockS3,
			}
		})

		It("should successfully delete a bucket", func(ctx SpecContext) {
			err := client.DeleteBucket(ctx, "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should handle errors when deleting a bucket", func(ctx SpecContext) {
			mockS3.DeleteBucketFunc = func(ctx context.Context, input *s3.DeleteBucketInput, opts ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
				return nil, fmt.Errorf("mock delete bucket error")
			}

			err := client.DeleteBucket(ctx, "test-bucket")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mock delete bucket error"))
		})
	})
})
