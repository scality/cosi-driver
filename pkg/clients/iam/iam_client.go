package iamclient

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/smithy-go/logging"
	"github.com/scality/cosi-driver/pkg/util"
	"k8s.io/klog/v2"
)

// postfix for inline policy which is created when COSI receives a BucketAccess (BA) request
const IAMUserInlinePolicyPostfix = "-cosi-ba"

type IAMAPI interface {
	CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
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
	policyName := fmt.Sprintf("%s%s", bucketName, IAMUserInlinePolicyPostfix)
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
		PolicyName:     &policyName,
		PolicyDocument: &policyDocument,
	}

	_, err := client.IAMService.PutUserPolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to attach inline policy to IAM user %s: %w", userName, err)
	}

	klog.InfoS("Inline policy attachment succeeded", "user", userName, "policyName", policyName)
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
