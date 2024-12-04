package iamclient_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	iamclient "github.com/scality/cosi-driver/pkg/clients/iam"
	"github.com/scality/cosi-driver/pkg/util"
)

// MockIAMClient implements the IAMAPI interface for testing
type MockIAMClient struct {
	CreateUserFunc      func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicyFunc   func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKeyFunc func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
}

func (m *MockIAMClient) CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, input, opts...)
	}
	return &iam.CreateUserOutput{}, nil
}

func (m *MockIAMClient) PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
	if m.PutUserPolicyFunc != nil {
		return m.PutUserPolicyFunc(ctx, input, opts...)
	}
	return &iam.PutUserPolicyOutput{}, nil
}

func (m *MockIAMClient) CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
	if m.CreateAccessKeyFunc != nil {
		return m.CreateAccessKeyFunc(ctx, input, opts...)
	}
	return &iam.CreateAccessKeyOutput{}, nil
}

func TestIAMClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IAMClient Test Suite")
}

var _ = Describe("IAMClient", func() {
	var params util.StorageClientParameters

	BeforeEach(func() {
		params = util.StorageClientParameters{
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			Endpoint:        "https://iam.mock.endpoint",
			Region:          "us-west-2",
			TLSCert:         nil,
			Debug:           false,
		}
	})

	Describe("IAM Operations", func() {
		var mockIAM *MockIAMClient

		BeforeEach(func() {
			mockIAM = &MockIAMClient{}
		})

		It("should successfully create a user", func(ctx SpecContext) {
			mockIAM.CreateUserFunc = func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				return &iam.CreateUserOutput{}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.CreateUser(ctx, "test-user")
			Expect(err).To(BeNil())
		})

		It("should return an error when CreateUser fails", func(ctx SpecContext) {
			mockIAM.CreateUserFunc = func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
				return nil, fmt.Errorf("simulated CreateUser failure")
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.CreateUser(ctx, "test-user")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to create IAM user test-user"))
			Expect(err.Error()).To(ContainSubstring("simulated CreateUser failure"))
		})

		It("should attach an inline policy successfully", func(ctx SpecContext) {
			mockIAM.PutUserPolicyFunc = func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				Expect(*input.PolicyDocument).To(ContainSubstring("s3:*"))
				Expect(*input.PolicyDocument).To(ContainSubstring("arn:aws:s3:::test-bucket"))
				return &iam.PutUserPolicyOutput{}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.AttachS3WildcardInlinePolicy(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should return an error when PutUserPolicy fails", func(ctx SpecContext) {
			mockIAM.PutUserPolicyFunc = func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
				return nil, fmt.Errorf("simulated PutUserPolicy failure")
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.AttachS3WildcardInlinePolicy(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to attach inline policy to IAM user test-user"))
			Expect(err.Error()).To(ContainSubstring("simulated PutUserPolicy failure"))
		})

		It("should create an access key successfully", func(ctx SpecContext) {
			mockIAM.CreateAccessKeyFunc = func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				return &iam.CreateAccessKeyOutput{
					AccessKey: &types.AccessKey{
						AccessKeyId:     aws.String("test-access-key-id"),
						SecretAccessKey: aws.String("test-secret-access-key"),
					},
				}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			output, err := client.CreateAccessKey(ctx, "test-user")
			Expect(err).To(BeNil())
			Expect(output.AccessKey.AccessKeyId).To(Equal(aws.String("test-access-key-id")))
			Expect(output.AccessKey.SecretAccessKey).To(Equal(aws.String("test-secret-access-key")))
		})

		It("should return an error when CreateAccessKey fails", func(ctx SpecContext) {
			mockIAM.CreateAccessKeyFunc = func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
				return nil, fmt.Errorf("simulated CreateAccessKey failure")
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			output, err := client.CreateAccessKey(ctx, "test-user")
			Expect(err).NotTo(BeNil())
			Expect(output).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to create access key for IAM user test-user"))
			Expect(err.Error()).To(ContainSubstring("simulated CreateAccessKey failure"))
		})
	})

	Describe("ConfigureTLSTransport", func() {
		It("should configure TLS when valid certData is provided", func() {
			certData := []byte("invalid-cert-data")
			transport := iamclient.ConfigureTLSTransport(certData, false)

			Expect(transport).NotTo(BeNil())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			Expect(transport.TLSClientConfig.RootCAs).NotTo(BeNil())
		})

		It("should log a warning if invalid certData is provided", func() {
			certData := []byte("invalid-cert-data")
			transport := iamclient.ConfigureTLSTransport(certData, false)

			Expect(transport).NotTo(BeNil())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			Expect(transport.TLSClientConfig.RootCAs).NotTo(BeNil())
		})

		It("should skip RootCAs configuration when no certData is provided", func() {
			transport := iamclient.ConfigureTLSTransport(nil, false)

			Expect(transport).NotTo(BeNil())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			Expect(transport.TLSClientConfig.RootCAs).To(BeNil())
		})
	})

	Describe("CreateBucketAccess", func() {
		var mockIAM *MockIAMClient

		BeforeEach(func() {
			mockIAM = &MockIAMClient{}
		})

		It("should successfully create a user, attach a policy, and generate an access key", func(ctx SpecContext) {
			mockIAM.CreateUserFunc = func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				return &iam.CreateUserOutput{}, nil
			}

			mockIAM.PutUserPolicyFunc = func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				Expect(*input.PolicyDocument).To(ContainSubstring("arn:aws:s3:::test-bucket"))
				return &iam.PutUserPolicyOutput{}, nil
			}

			mockIAM.CreateAccessKeyFunc = func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				return &iam.CreateAccessKeyOutput{
					AccessKey: &types.AccessKey{
						AccessKeyId:     aws.String("test-access-key-id"),
						SecretAccessKey: aws.String("test-secret-access-key"),
					},
				}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			output, err := client.CreateBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
			Expect(output).NotTo(BeNil())
			Expect(output.AccessKey.AccessKeyId).To(Equal(aws.String("test-access-key-id")))
			Expect(output.AccessKey.SecretAccessKey).To(Equal(aws.String("test-secret-access-key")))
		})

		It("should return an error if CreateUser fails", func(ctx SpecContext) {
			mockIAM.CreateUserFunc = func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
				return nil, fmt.Errorf("simulated CreateUser failure")
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			output, err := client.CreateBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(output).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("simulated CreateUser failure"))
		})

		It("should return an error if AttachS3WildcardInlinePolicy fails", func(ctx SpecContext) {
			mockIAM.CreateUserFunc = func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
				return &iam.CreateUserOutput{}, nil
			}

			mockIAM.PutUserPolicyFunc = func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
				return nil, fmt.Errorf("simulated AttachS3WildcardInlinePolicy failure")
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			output, err := client.CreateBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(output).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("simulated AttachS3WildcardInlinePolicy failure"))
		})

		It("should return an error if CreateAccessKey fails", func(ctx SpecContext) {
			mockIAM.CreateUserFunc = func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
				return &iam.CreateUserOutput{}, nil
			}

			mockIAM.PutUserPolicyFunc = func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
				return &iam.PutUserPolicyOutput{}, nil
			}

			mockIAM.CreateAccessKeyFunc = func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
				return nil, fmt.Errorf("simulated CreateAccessKey failure")
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			output, err := client.CreateBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(output).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("simulated CreateAccessKey failure"))
		})
	})

})
