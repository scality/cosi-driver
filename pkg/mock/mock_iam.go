package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// MockIAMClient simulates the behavior of an IAM client for testing purposes.
type MockIAMClient struct {
	CreateUserFunc       func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicyFunc    func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKeyFunc  func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	GetUserFunc          func(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error)
	DeleteUserPolicyFunc func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error)
	ListAccessKeysFunc   func(ctx context.Context, input *iam.ListAccessKeysInput, opts ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	DeleteAccessKeyFunc  func(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	DeleteUserFunc       func(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error)
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

// GetUser retrieves a mock IAM user.
func (m *MockIAMClient) GetUser(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, input, opts...)
	}
	return &iam.GetUserOutput{
		User: &types.User{
			UserName: input.UserName,
			UserId:   aws.String("mock-user-id"),
		},
	}, nil
}

// DeleteUserPolicy deletes a mock inline policy for the user.
func (m *MockIAMClient) DeleteUserPolicy(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
	if m.DeleteUserPolicyFunc != nil {
		return m.DeleteUserPolicyFunc(ctx, input, opts...)
	}
	return &iam.DeleteUserPolicyOutput{}, nil
}

// ListAccessKeys retrieves mock access keys for the user.
func (m *MockIAMClient) ListAccessKeys(ctx context.Context, input *iam.ListAccessKeysInput, opts ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	if m.ListAccessKeysFunc != nil {
		return m.ListAccessKeysFunc(ctx, input, opts...)
	}
	return &iam.ListAccessKeysOutput{
		AccessKeyMetadata: []types.AccessKeyMetadata{
			{AccessKeyId: aws.String("mock-access-key-id")},
		},
	}, nil
}

// DeleteAccessKey deletes a mock access key for the user.
func (m *MockIAMClient) DeleteAccessKey(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
	if m.DeleteAccessKeyFunc != nil {
		return m.DeleteAccessKeyFunc(ctx, input, opts...)
	}
	return &iam.DeleteAccessKeyOutput{}, nil
}

// DeleteUser deletes a mock IAM user.
func (m *MockIAMClient) DeleteUser(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, input, opts...)
	}
	return &iam.DeleteUserOutput{}, nil
}
