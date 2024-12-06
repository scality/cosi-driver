package iamclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go/logging"
	"github.com/scality/cosi-driver/pkg/util"
	"k8s.io/klog/v2"
)

type IAMAPI interface {
	CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	GetUser(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error)
	DeleteUserPolicy(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error)
	ListAccessKeys(ctx context.Context, input *iam.ListAccessKeysInput, opts ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	DeleteAccessKey(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	DeleteUser(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error)
}

type IAMClient struct {
	IAMService IAMAPI
}

func InitIAMClient(params util.StorageClientParameters) (*IAMClient, error) {
	var logger logging.Logger
	if params.Debug {
		logger = logging.NewStandardLogger(os.Stdout)
	} else {
		logger = nil
	}

	httpClient := &http.Client{
		Timeout: util.DefaultRequestTimeout,
	}

	if strings.HasPrefix(params.IAMEndpoint, "https://") {
		httpClient.Transport = util.ConfigureTLSTransport(params.TLSCert)
	}

	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(params.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKeyID, params.SecretAccessKey, "")),
		config.WithHTTPClient(httpClient),
		config.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	iamClient := iam.NewFromConfig(awsCfg, func(o *iam.Options) {
		o.BaseEndpoint = &params.IAMEndpoint
	})

	return &IAMClient{
		IAMService: iamClient,
	}, nil
}

// CreateUser creates an IAM user with the specified name.
func (client *IAMClient) CreateUser(ctx context.Context, userName string) error {
	input := &iam.CreateUserInput{
		UserName: &userName,
	}

	_, err := client.IAMService.CreateUser(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create IAM user %s: %w", userName, err)
	}

	klog.InfoS("IAM user creation succeeded", "user", userName)
	return nil
}

// AttachS3WildcardInlinePolicy attaches an inline policy to an IAM user for a specific bucket.
func (client *IAMClient) AttachS3WildcardInlinePolicy(ctx context.Context, userName, bucketName string) error {
	policyDocument := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": "s3:*",
				"Resource": [
					"arn:aws:s3:::%s",
					"arn:aws:s3:::%s/*"
				]
			}
		]
	}`, bucketName, bucketName)

	input := &iam.PutUserPolicyInput{
		UserName:       &userName,
		PolicyName:     &bucketName,
		PolicyDocument: &policyDocument,
	}

	_, err := client.IAMService.PutUserPolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to attach inline policy to IAM user %s: %w", userName, err)
	}

	klog.InfoS("Inline policy attachment succeeded", "user", userName, "policyName", bucketName)
	return nil
}

// CreateAccessKey generates access keys for an IAM user.
func (client *IAMClient) CreateAccessKey(ctx context.Context, userName string) (*iam.CreateAccessKeyOutput, error) {
	input := &iam.CreateAccessKeyInput{
		UserName: &userName,
	}

	output, err := client.IAMService.CreateAccessKey(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create access key for IAM user %s: %w", userName, err)
	}

	klog.InfoS("Access key creation succeeded", "user", userName)
	return output, nil
}

// CreateBucketAccess is a helper that combines user creation, policy attachment, and access key generation.
func (client *IAMClient) CreateBucketAccess(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error) {
	err := client.CreateUser(ctx, userName)
	if err != nil {
		return nil, err
	}

	err = client.AttachS3WildcardInlinePolicy(ctx, userName, bucketName)
	if err != nil {
		return nil, err
	}

	accessKeyOutput, err := client.CreateAccessKey(ctx, userName)
	if err != nil {
		return nil, err
	}

	return accessKeyOutput, nil
}

// RevokeBucketAccess revokes bucket access for a user by performing the following:
// 1. Checks if the user exists.
// 2. Deletes the inline policy associated with the bucket (if it exists).
// 3. Lists and deletes all access keys for the user.
// 4. Deletes the user itself.
func (client *IAMClient) RevokeBucketAccess(ctx context.Context, userName, bucketName string) error {
	_, err := client.IAMService.GetUser(ctx, &iam.GetUserInput{UserName: &userName})
	if err != nil {
		var noSuchEntityErr *types.NoSuchEntityException
		if errors.As(err, &noSuchEntityErr) {
			klog.InfoS("IAM user does not exist, nothing to revoke", "user", userName)
			return nil // User doesn't exist, no action needed
		}
		return fmt.Errorf("failed to get IAM user %s: %w", userName, err)
	}

	_, err = client.IAMService.DeleteUserPolicy(ctx, &iam.DeleteUserPolicyInput{
		UserName:   &userName,
		PolicyName: &bucketName, // Assuming the policy name is the same as the bucket name
	})
	if err != nil {
		var noSuchEntityErr *types.NoSuchEntityException
		if !errors.As(err, &noSuchEntityErr) {
			// Log and proceed if the policy doesn't exist
			return fmt.Errorf("failed to delete inline policy %s for user %s: %w", bucketName, userName, err)
		}
		klog.V(3).InfoS("Inline policy does not exist, skipping deletion", "user", userName, "policyName", bucketName)
	} else {
		klog.InfoS("Successfully deleted inline policy", "user", userName, "policyName", bucketName)
	}

	listKeysOutput, err := client.IAMService.ListAccessKeys(ctx, &iam.ListAccessKeysInput{UserName: &userName})
	if err != nil {
		return fmt.Errorf("failed to list access keys for IAM user %s: %w", userName, err)
	}

	for _, key := range listKeysOutput.AccessKeyMetadata {
		_, err := client.IAMService.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
			UserName:    &userName,
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			return fmt.Errorf("failed to delete access key %s for IAM user %s: %w", *key.AccessKeyId, userName, err)
		}
		klog.InfoS("Successfully deleted access key", "user", userName, "accessKeyId", *key.AccessKeyId)
	}

	_, err = client.IAMService.DeleteUser(ctx, &iam.DeleteUserInput{UserName: &userName})
	if err != nil {
		var noSuchEntityErr *types.NoSuchEntityException
		if !errors.As(err, &noSuchEntityErr) {
			return fmt.Errorf("failed to delete IAM user %s: %w", userName, err)
		}
		klog.InfoS("IAM user does not exist, skipping deletion", "user", userName)
	} else {
		klog.InfoS("Successfully deleted IAM user", "user", userName)
	}

	return nil
}
