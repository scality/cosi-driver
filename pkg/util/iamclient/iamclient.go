package iamclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/smithy-go/logging"
	"k8s.io/klog/v2"
)

const (
	defaultRegion  = "us-east-1"
	requestTimeout = 15 * time.Second
)

type IAMAPI interface {
	CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
}

type IAMParams struct {
	AccessKey string
	SecretKey string
	Region    string
	Debug     bool
}

type IAMClient struct {
	IAMService IAMAPI
}

// InitIAMClient initializes the IAM client.
func InitIAMClient(params IAMParams) (*IAMClient, error) {
	if params.AccessKey == "" || params.SecretKey == "" {
		return nil, fmt.Errorf("AWS credentials are missing")
	}

	var logger logging.Logger
	if params.Debug {
		logger = logging.NewStandardLogger(os.Stdout)
	} else {
		logger = nil
	}

	ctx := context.Background()

	region := params.Region
	if region == "" {
		region = defaultRegion
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKey, params.SecretKey, "")),
		config.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	iamClient := iam.NewFromConfig(awsCfg)

	return &IAMClient{
		IAMService: iamClient,
	}, nil
}

// CreateUser creates an IAM user.
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

// GenerateBucketInlinePolicyDocument generates a policy JSON for S3 bucket access.
func GenerateBucketInlinePolicyDocument(bucketName string) (string, error) {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": "s3:*",
				"Resource": []string{
					fmt.Sprintf("arn:aws:s3:::%s", bucketName),
					fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
				},
			},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to generate policy JSON: %w", err)
	}

	return string(policyJSON), nil
}

// AttachInlinePolicy attaches an inline policy to an IAM user.
func (client *IAMClient) AttachInlinePolicy(ctx context.Context, userName, policyName, bucketName string) error {
	policyDocument, err := GenerateBucketInlinePolicyDocument(bucketName)
	if err != nil {
		return err
	}

	input := &iam.PutUserPolicyInput{
		UserName:       &userName,
		PolicyName:     &policyName,
		PolicyDocument: &policyDocument,
	}

	_, err = client.IAMService.PutUserPolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to attach inline policy %s to IAM user %s: %w", policyName, userName, err)
	}

	klog.InfoS("Inline policy attachment succeeded", "user", userName, "policy", policyName)
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

// CreateUserWithPolicyAndAccessKey creates a user, attaches a bucket policy, and generates access keys.
func (client *IAMClient) CreateUserWithPolicyAndAccessKey(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error) {
	err := client.CreateUser(ctx, userName)
	if err != nil {
		return nil, err
	}

	policyName := fmt.Sprintf("%s-bucket-access", bucketName)
	err = client.AttachInlinePolicy(ctx, userName, policyName, bucketName)
	if err != nil {
		return nil, err
	}

	accessKeyOutput, err := client.CreateAccessKey(ctx, userName)
	if err != nil {
		return nil, err
	}

	return accessKeyOutput, nil
}
