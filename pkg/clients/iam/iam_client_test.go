package iamclient_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/smithy-go"
	iamclient "github.com/scality/cosi-driver/pkg/clients/iam"
	"github.com/scality/cosi-driver/pkg/mock"
	"github.com/scality/cosi-driver/pkg/util"
)

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
			IAMEndpoint:     "https://iam.mock.endpoint",
		}
	})

	Describe("IAM Operations", func() {
		var mockIAM *mock.MockIAMClient

		BeforeEach(func() {
			mockIAM = &mock.MockIAMClient{}
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

		It("should attach an inline policy with the correct name and content", func(ctx SpecContext) {
			bucketName := "inline-policy-bucket-test"
			mockIAM.PutUserPolicyFunc = func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
				expectedPolicyName := bucketName
				Expect(input.UserName).To(Equal(aws.String("test-user")))
				Expect(*input.PolicyName).To(Equal(expectedPolicyName))
				Expect(*input.PolicyDocument).To(ContainSubstring("s3:*"))
				Expect(*input.PolicyDocument).To(ContainSubstring(fmt.Sprintf("arn:aws:s3:::%s", bucketName)))
				return &iam.PutUserPolicyOutput{}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.AttachS3WildcardInlinePolicy(ctx, "test-user", bucketName)
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

	Describe("CreateBucketAccess", func() {
		var mockIAM *mock.MockIAMClient

		BeforeEach(func() {
			mockIAM = &mock.MockIAMClient{}
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

	Describe("InitIAMClient", func() {
		It("should return an error if AWS config loading fails", func() {
			originalLoadAWSConfig := iamclient.LoadAWSConfig
			defer func() { iamclient.LoadAWSConfig = originalLoadAWSConfig }()

			iamclient.LoadAWSConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
				return aws.Config{}, fmt.Errorf("mock LoadAWSConfig failure")
			}

			client, err := iamclient.InitIAMClient(params)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load AWS config: mock LoadAWSConfig failure"))
			Expect(client).To(BeNil())
		})

		It("should set up a logger when Debug is enabled", func() {
			params.Debug = true

			// Mock LoadAWSConfig
			originalLoadAWSConfig := iamclient.LoadAWSConfig
			defer func() { iamclient.LoadAWSConfig = originalLoadAWSConfig }()

			var loggerUsed bool
			iamclient.LoadAWSConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
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

			_, err := iamclient.InitIAMClient(params)
			Expect(err).To(BeNil())
			Expect(loggerUsed).To(BeTrue(), "Expected logger to be used when Debug is enabled")
		})
	})

	Describe("RevokeBucketAccess", func() {
		var mockIAM *mock.MockIAMClient
		noSuchEntityError := &types.NoSuchEntityException{}
		accessDeniedError := &smithy.GenericAPIError{
			Code:    "AccessDenied",
			Message: "Access Denied",
		}

		BeforeEach(func() {
			mockIAM = &mock.MockIAMClient{}
		})

		It("should fail on non-existent user gracefully", func(ctx SpecContext) {

			mockIAM.GetUserFunc = func(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error) {
				return nil, noSuchEntityError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "non-existent-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to get IAM user non-existent-user"))
			Expect(errors.As(err, &noSuchEntityError)).To(BeTrue())
		})

		It("should return an error if getting user fails", func(ctx SpecContext) {
			mockIAM.GetUserFunc = func(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error) {
				return nil, noSuchEntityError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(errors.As(err, &noSuchEntityError)).To(BeTrue())
		})

		It("should skip deletion if inline policy does not exist", func(ctx SpecContext) {
			mockIAM.DeleteUserPolicyFunc = func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
				return nil, noSuchEntityError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should return an error if deleting inline policy fails", func(ctx SpecContext) {
			mockIAM.DeleteUserPolicyFunc = func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
				return nil, accessDeniedError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(errors.As(err, &accessDeniedError)).To(BeTrue())
		})

		It("should not return an error if inline policy is not found", func(ctx SpecContext) {
			mockIAM.DeleteUserPolicyFunc = func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
				return nil, &types.NoSuchEntityException{}
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should successfully delete all access keys for the user", func(ctx SpecContext) {
			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should return an error if deleting access key fails", func(ctx SpecContext) {
			mockIAM.DeleteAccessKeyFunc = func(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
				return nil, accessDeniedError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(errors.As(err, &accessDeniedError)).To(BeTrue())
		})

		It("should not return an error if access key is not found", func(ctx SpecContext) {
			mockIAM.DeleteAccessKeyFunc = func(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
				return nil, noSuchEntityError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should successfully delete the user", func(ctx SpecContext) {
			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should return an error if deleting user fails", func(ctx SpecContext) {
			mockIAM.DeleteUserFunc = func(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
				return nil, accessDeniedError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.RevokeBucketAccess(ctx, "test-user", "test-bucket")
			Expect(err).NotTo(BeNil())
			Expect(errors.As(err, &accessDeniedError)).To(BeTrue())
		})

		It("should successfully delete an inline policy", func(ctx SpecContext) {
			mockIAM.DeleteUserPolicyFunc = func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
				Expect(*input.PolicyName).To(Equal("test-bucket"))
				Expect(*input.UserName).To(Equal("test-user"))
				return &iam.DeleteUserPolicyOutput{}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.DeleteInlinePolicy(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should skip deletion if inline policy does not exist", func(ctx SpecContext) {
			mockIAM.DeleteUserPolicyFunc = func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
				return nil, &types.NoSuchEntityException{}
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.DeleteInlinePolicy(ctx, "test-user", "test-bucket")
			Expect(err).To(BeNil())
		})

		It("should successfully delete all access keys", func(ctx SpecContext) {
			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.DeleteAllAccessKeys(ctx, "test-user")
			Expect(err).To(BeNil())
		})

		It("should successfully delete a user", func(ctx SpecContext) {
			mockIAM.DeleteUserFunc = func(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
				Expect(*input.UserName).To(Equal("test-user"))
				return &iam.DeleteUserOutput{}, nil
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.DeleteUser(ctx, "test-user")
			Expect(err).To(BeNil())
		})

		It("should skip deletion if user does not exist", func(ctx SpecContext) {
			mockIAM.DeleteUserFunc = func(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
				return nil, &types.NoSuchEntityException{}
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.DeleteUser(ctx, "test-user")
			Expect(err).To(BeNil())
		})

		It("should return an error if listing access keys fails", func(ctx SpecContext) {
			mockIAM.ListAccessKeysFunc = func(ctx context.Context, input *iam.ListAccessKeysInput, opts ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
				return nil, accessDeniedError
			}

			client, _ := iamclient.InitIAMClient(params)
			client.IAMService = mockIAM

			err := client.DeleteAllAccessKeys(ctx, "test-user")
			Expect(err).NotTo(BeNil())
			Expect(errors.As(err, &accessDeniedError)).To(BeTrue())
		})
	})
})
