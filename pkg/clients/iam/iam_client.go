package iamclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go/logging"
	c "github.com/scality/cosi-driver/pkg/constants"
	"github.com/scality/cosi-driver/pkg/metrics"
	"github.com/scality/cosi-driver/pkg/util"
	"k8s.io/klog/v2"
)

// postfix for inline policy which is created when COSI receives a BucketAccess (BA) request
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

var LoadAWSConfig = config.LoadDefaultConfig

var InitIAMClient = func(params util.StorageClientParameters) (*IAMClient, error) {
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
		klog.V(c.LvlDebug).InfoS("Configuring TLS transport for IAM client", "IAMEndpoint", params.IAMEndpoint)
		httpClient.Transport = util.ConfigureTLSTransport(params.TLSCert)
	}

	ctx := context.Background()

	awsCfg, err := LoadAWSConfig(ctx,
		config.WithRegion(params.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKeyID, params.SecretAccessKey, "")),
		config.WithHTTPClient(httpClient),
		config.WithLogger(logger),
	)
	if err != nil {
		return nil, err
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
	method := "CreateUser"
	start := time.Now()

	input := &iam.CreateUserInput{
		UserName: &userName,
	}
	_, err := client.IAMService.CreateUser(ctx, input)
	duration := time.Since(start).Seconds()
	status := c.StatusSuccess
	if err != nil {
		status = c.StatusError
	}

	metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
	metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
	return err
}

// AttachS3WildcardInlinePolicy attaches an inline policy to an IAM user for a specific bucket.
func (client *IAMClient) AttachS3WildcardInlinePolicy(ctx context.Context, userName, bucketName string) error {
	method := "PutUserPolicy"
	start := time.Now()

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
	duration := time.Since(start).Seconds()
	status := c.StatusSuccess
	if err != nil {
		status = c.StatusError
	}

	metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
	metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
	return err
}

// CreateAccessKey generates access keys for an IAM user.
func (client *IAMClient) CreateAccessKey(ctx context.Context, userName string) (*iam.CreateAccessKeyOutput, error) {
	method := "CreateAccessKey"
	start := time.Now()

	input := &iam.CreateAccessKeyInput{
		UserName: &userName,
	}

	output, err := client.IAMService.CreateAccessKey(ctx, input)
	duration := time.Since(start).Seconds()
	status := c.StatusSuccess
	if err != nil {
		status = c.StatusError
	}

	metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
	metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
	return output, err
}

// CreateBucketAccess is a helper that combines user creation, policy attachment, and access key generation.
func (client *IAMClient) CreateBucketAccess(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error) {
	err := client.CreateUser(ctx, userName)
	if err != nil {
		return nil, err
	}
	klog.V(c.LvlInfo).InfoS("Successfully created IAM user", "userName", userName)

	err = client.AttachS3WildcardInlinePolicy(ctx, userName, bucketName)
	if err != nil {
		return nil, err
	}
	klog.V(c.LvlInfo).InfoS("Successfully attached inline policy", "userName", userName, "policyName", bucketName)

	accessKeyOutput, err := client.CreateAccessKey(ctx, userName)
	if err != nil {
		return nil, err
	}
	klog.V(c.LvlInfo).InfoS("Successfully created access key", "userName", userName)

	return accessKeyOutput, nil
}

// RevokeBucketAccess is a helper that revokes bucket access by orchestrating individual steps to delete the user, inline policy, and access keys.
func (client *IAMClient) RevokeBucketAccess(ctx context.Context, userName, bucketName string) error {
	err := client.EnsureUserExists(ctx, userName)
	if err != nil {
		return err
	}
	klog.V(c.LvlInfo).InfoS("Verified IAM user exists", "userName", userName)

	err = client.DeleteInlinePolicy(ctx, userName, bucketName)
	if err != nil {
		return err
	}
	klog.V(c.LvlInfo).InfoS("Deleted inline policy if it existed", "userName", userName, "policyName", bucketName)

	err = client.DeleteAllAccessKeys(ctx, userName)
	if err != nil {
		return err
	}
	klog.V(c.LvlInfo).InfoS("Deleted all access keys if any existed", "userName", userName)

	err = client.DeleteUser(ctx, userName)
	if err != nil {
		return err
	}
	klog.V(c.LvlInfo).InfoS("Deleted IAM user", "userName", userName)
	return nil
}

func (client *IAMClient) EnsureUserExists(ctx context.Context, userName string) error {
	method := "GetUser"
	start := time.Now()

	_, err := client.IAMService.GetUser(ctx, &iam.GetUserInput{UserName: &userName})
	duration := time.Since(start).Seconds()
	status := c.StatusSuccess
	if err != nil {
		status = c.StatusError
	}

	metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
	metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
	return err
}

func (client *IAMClient) DeleteInlinePolicy(ctx context.Context, userName, bucketName string) error {
	method := "DeleteInlinePolicy"
	start := time.Now()

	_, err := client.IAMService.DeleteUserPolicy(ctx, &iam.DeleteUserPolicyInput{
		UserName:   &userName,
		PolicyName: &bucketName,
	})
	duration := time.Since(start).Seconds()

	status := c.StatusSuccess
	if err != nil {
		status = c.StatusError
		metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
		metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
		var noSuchEntityErr *types.NoSuchEntityException
		if errors.As(err, &noSuchEntityErr) {
			klog.V(c.LvlDebug).InfoS("Inline policy does not exist, skipping deletion", "user", userName, "policyName", bucketName)
			return nil
		}
		return err
	}

	metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
	metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
	klog.V(c.LvlDebug).InfoS("Successfully deleted inline policy", "userName", userName, "policyName", bucketName)
	return nil
}

func (client *IAMClient) DeleteAllAccessKeys(ctx context.Context, userName string) error {
	listMethod := "ListAccessKeys"
	listStart := time.Now()
	listKeysOutput, err := client.IAMService.ListAccessKeys(ctx, &iam.ListAccessKeysInput{UserName: &userName})
	listDuration := time.Since(listStart).Seconds()

	listStatus := c.StatusSuccess
	if err != nil {
		listStatus = c.StatusError
		metrics.IAMRequestsTotal.WithLabelValues(listMethod, listStatus).Inc()
		metrics.IAMRequestDuration.WithLabelValues(listMethod, listStatus).Observe(listDuration)
		return err
	}
	metrics.IAMRequestsTotal.WithLabelValues(listMethod, listStatus).Inc()
	metrics.IAMRequestDuration.WithLabelValues(listMethod, listStatus).Observe(listDuration)

	deleteKeyMethod := "DeleteAccessKey"
	var noSuchEntityErr *types.NoSuchEntityException
	for _, key := range listKeysOutput.AccessKeyMetadata {
		klog.V(c.LvlTrace).InfoS("Deleting access key", "userName", userName, "accessKeyId", *key.AccessKeyId)
		deleteStart := time.Now()
		_, err := client.IAMService.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
			UserName:    &userName,
			AccessKeyId: key.AccessKeyId,
		})
		deleteDuration := time.Since(deleteStart).Seconds()

		deleteStatus := c.StatusSuccess
		if err != nil {
			deleteStatus = c.StatusError
			metrics.IAMRequestsTotal.WithLabelValues(deleteKeyMethod, deleteStatus).Inc()
			metrics.IAMRequestDuration.WithLabelValues(deleteKeyMethod, deleteStatus).Observe(deleteDuration)
			if errors.As(err, &noSuchEntityErr) {
				klog.V(c.LvlTrace).InfoS("Access key does not exist, skipping deletion", "userName", userName, "accessKeyId", *key.AccessKeyId)
				continue
			}
			return err
		}
		metrics.IAMRequestsTotal.WithLabelValues(deleteKeyMethod, deleteStatus).Inc()
		metrics.IAMRequestDuration.WithLabelValues(deleteKeyMethod, deleteStatus).Observe(deleteDuration)
		klog.V(c.LvlTrace).InfoS("Successfully deleted access key", "userName", userName, "accessKeyId", *key.AccessKeyId)
	}
	klog.V(c.LvlDebug).InfoS("Successfully deleted all access keys", "userName", userName)
	return nil
}

func (client *IAMClient) DeleteUser(ctx context.Context, userName string) error {
	method := "DeleteUser"
	start := time.Now()

	_, err := client.IAMService.DeleteUser(ctx, &iam.DeleteUserInput{UserName: &userName})
	duration := time.Since(start).Seconds()

	status := c.StatusSuccess
	if err != nil {
		status = c.StatusError
		metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
		metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)

		var noSuchEntityErr *types.NoSuchEntityException
		if errors.As(err, &noSuchEntityErr) {
			// For scenarios where the user has been manually deleted between the GetUser and DeleteUser requests.
			// Since the IAM endpoint and existence check were correct earlier, it is safe to skip deletion
			// and treat this as a non-critical error.
			klog.InfoS("IAM user does not exist, skipping deletion", "user", userName)
			return nil
		}
		return err
	}

	metrics.IAMRequestsTotal.WithLabelValues(method, status).Inc()
	metrics.IAMRequestDuration.WithLabelValues(method, status).Observe(duration)
	return nil
}
