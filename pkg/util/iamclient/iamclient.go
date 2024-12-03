package iamclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/smithy-go/logging"
	types "github.com/scality/cosi-driver/pkg/util/types"
	"k8s.io/klog/v2"
)

type IAMAPI interface {
	CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
}

const (
	defaultRegion  = "us-east-1"
	requestTimeout = 15 * time.Second
)

type IAMClient struct {
	IAMService IAMAPI
}

func InitClient(params types.StorageClientParameters) (*IAMClient, error) {
	if params.AccessKey == "" || params.SecretKey == "" {
		return nil, fmt.Errorf("AWS credentials are missing")
	}

	var logger logging.Logger
	if params.Debug {
		logger = logging.NewStandardLogger(os.Stdout)
	} else {
		logger = nil
	}

	httpClient := &http.Client{
		Timeout: requestTimeout,
	}

	// in the case where endpoint is HTTPS but no certificate is provided, skip TLS validation
	isHTTPSEndpoint := strings.HasPrefix(params.Endpoint, "https://")
	skipTLSValidation := isHTTPSEndpoint && len(params.TLSCert) == 0
	if isHTTPSEndpoint {
		httpClient.Transport = ConfigureTLSTransport(params.TLSCert, skipTLSValidation)
	}

	region := params.Region
	if region == "" {
		region = defaultRegion
	}

	ctx := context.Background()

	awsCfg, err := awssdkconfig.LoadDefaultConfig(ctx,
		awssdkconfig.WithRegion(region),
		awssdkconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKey, params.SecretKey, "")),
		awssdkconfig.WithHTTPClient(httpClient),
		awssdkconfig.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	iamClient := iam.NewFromConfig(awsCfg, func(o *iam.Options) {
		o.BaseEndpoint = &params.Endpoint
	})

	return &IAMClient{
		IAMService: iamClient,
	}, nil
}

func ConfigureTLSTransport(certData []byte, skipTLSValidation bool) *http.Transport {
	tlsSettings := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: skipTLSValidation,
	}

	if len(certData) > 0 {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certData); !ok {
			klog.Warning("Failed to append provided cert data to the certificate pool")
		}
		tlsSettings.RootCAs = caCertPool
	}

	return &http.Transport{
		TLSClientConfig: tlsSettings,
	}
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

// AttachInlinePolicy attaches an inline policy to an IAM user for a specific bucket.
func (client *IAMClient) AttachInlinePolicy(ctx context.Context, userName, bucketName string) error {
	policyName := fmt.Sprintf("%s-cosi-ba", bucketName)
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

	err = client.AttachInlinePolicy(ctx, userName, bucketName)
	if err != nil {
		return nil, err
	}

	accessKeyOutput, err := client.CreateAccessKey(ctx, userName)
	if err != nil {
		return nil, err
	}

	return accessKeyOutput, nil
}
