package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// MockIAMClient simulates the behavior of an IAM client for testing purposes.
// It embeds iamclient.IAMClient to ensure compatibility with the interface or struct.
type MockIAMClient struct {
	CreateUserFunc      func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicyFunc   func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKeyFunc func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
}

// CreateUser creates a mock IAM user with default behavior or custom logic.
func (m *MockIAMClient) CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, input, opts...)
	}
	return &iam.CreateUserOutput{
		User: &types.User{
			UserName: input.UserName,
			UserId:   aws.String("mock-user-id"),
		},
	}, nil
}

// PutUserPolicy attaches a mock inline policy to the user.
func (m *MockIAMClient) PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
	if m.PutUserPolicyFunc != nil {
		return m.PutUserPolicyFunc(ctx, input, opts...)
	}
	return &iam.PutUserPolicyOutput{}, nil
}

// CreateAccessKey generates a mock access key for the user.
func (m *MockIAMClient) CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
	if m.CreateAccessKeyFunc != nil {
		return m.CreateAccessKeyFunc(ctx, input, opts...)
	}
	return &iam.CreateAccessKeyOutput{
		AccessKey: &types.AccessKey{
			AccessKeyId:     aws.String("mock-access-key-id"),
			SecretAccessKey: aws.String("mock-secret-access-key"),
		},
	}, nil
}
