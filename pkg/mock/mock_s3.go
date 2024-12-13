package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// MockS3Client simulates the behavior of an S3 client for testing.
type MockS3Client struct {
	CreateBucketFunc func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteBucketFunc func(ctx context.Context, input *s3.DeleteBucketInput, opts ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

// CreateBucket executes the mock CreateBucketFunc if defined, otherwise returns a default response.
func (m *MockS3Client) CreateBucket(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	if m.CreateBucketFunc != nil {
		return m.CreateBucketFunc(ctx, input, opts...)
	}
	return &s3.CreateBucketOutput{}, nil
}

// DeleteBucket executes the mock DeleteBucketFunc if defined, otherwise returns a default response.
func (m *MockS3Client) DeleteBucket(ctx context.Context, input *s3.DeleteBucketInput, opts ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	if m.DeleteBucketFunc != nil {
		return m.DeleteBucketFunc(ctx, input, opts...)
	}
	return &s3.DeleteBucketOutput{}, nil
}
