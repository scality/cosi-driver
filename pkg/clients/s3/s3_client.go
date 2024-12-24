package s3client

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
	"github.com/aws/smithy-go/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scality/cosi-driver/pkg/metrics"
	"github.com/scality/cosi-driver/pkg/util"
	"k8s.io/klog/v2"
)

// S3API defines the methods the S3 client must implement.
type S3API interface {
	CreateBucket(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteBucket(ctx context.Context, input *s3.DeleteBucketInput, opts ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

// S3Client wraps the S3 service client for custom operations and middleware integration.
type S3Client struct {
	S3Service S3API
}

// LoadAWSConfig is a wrapper for AWS SDK's default configuration loader.
var LoadAWSConfig = config.LoadDefaultConfig

// AttachPrometheusMiddleware attaches middleware to track metrics using Prometheus.
func AttachPrometheusMiddleware(stack *middleware.Stack) error {
	// Define the middleware logic
	middlewareFunc := middleware.FinalizeMiddlewareFunc("PrometheusMetrics", func(
		ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler,
	) (out middleware.FinalizeOutput, metadata middleware.Metadata, err error) {
		operationName := middleware.GetOperationName(ctx)

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(duration float64) {
			status := "success"
			if err != nil {
				status = "error"
			}
			metrics.S3RequestDuration.WithLabelValues(operationName, status).Observe(duration)
			metrics.S3RequestsTotal.WithLabelValues(operationName, status).Inc()
		}))
		defer timer.ObserveDuration()

		out, metadata, err = next.HandleFinalize(ctx, in)
		if err != nil {
			klog.ErrorS(err, "AWS SDK operation failed", "operation", operationName)
		}
		return out, metadata, err
	})

	// Add the middleware to the Finalize step
	return stack.Finalize.Add(middlewareFunc, middleware.After)
}

// InitS3Client initializes the S3 client with Prometheus middleware and custom configuration.
var InitS3Client = func(ctx context.Context, params util.StorageClientParameters) (*S3Client, error) {
	// Configure a logger
	var logger logging.Logger
	if params.Debug {
		logger = logging.NewStandardLogger(os.Stdout)
	} else {
		logger = nil
	}

	// Configure HTTP client with TLS support if needed
	httpClient := &http.Client{
		Timeout: util.DefaultRequestTimeout,
	}
	if strings.HasPrefix(params.Endpoint, "https://") {
		httpClient.Transport = util.ConfigureTLSTransport(params.TLSCert)
	}

	// Load AWS configuration with middleware
	awsCfg, err := LoadAWSConfig(ctx,
		config.WithRegion(params.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKeyID, params.SecretAccessKey, "")),
		config.WithHTTPClient(httpClient),
		config.WithLogger(logger),
		config.WithAPIOptions([]func(*middleware.Stack) error{
			AttachPrometheusMiddleware,
		}),
	)
	if err != nil {
		return nil, err
	}

	// Create the S3 client
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(params.Endpoint)
	})

	return &S3Client{
		S3Service: s3Client,
	}, nil
}

// CreateBucket creates a new bucket in the S3 service.
func (client *S3Client) CreateBucket(ctx context.Context, bucketName string, params util.StorageClientParameters) error {
	input := &s3.CreateBucketInput{Bucket: &bucketName}
	if params.Region != util.DefaultRegion {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(params.Region),
		}
	}

	_, err := client.S3Service.CreateBucket(ctx, input)
	return err // Metrics are handled by middleware
}

// DeleteBucket deletes a bucket in the S3 service.
func (client *S3Client) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := client.S3Service.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &bucketName})
	return err // Metrics are handled by middleware
}
