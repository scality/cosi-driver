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
	"github.com/scality/cosi-driver/pkg/metrics"
	"github.com/scality/cosi-driver/pkg/util"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

type S3API interface {
	CreateBucket(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteBucket(ctx context.Context, input *s3.DeleteBucketInput, opts ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

type S3Client struct {
	S3Service S3API
}

var LoadAWSConfig = config.LoadDefaultConfig

var InitS3Client = func(ctx context.Context, params util.StorageClientParameters) (*S3Client, error) {
	var logger logging.Logger
	if params.Debug {
		logger = logging.NewStandardLogger(os.Stdout)
	} else {
		logger = nil
	}

	httpClient := &http.Client{
		Timeout: util.DefaultRequestTimeout,
	}

	if strings.HasPrefix(params.Endpoint, "https://") {
		httpClient.Transport = util.ConfigureTLSTransport(params.TLSCert)
	}

	awsCfg, err := LoadAWSConfig(ctx,
		config.WithRegion(params.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKeyID, params.SecretAccessKey, "")),
		config.WithHTTPClient(httpClient),
		config.WithLogger(logger),
		config.WithAPIOptions([]func(*middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return metrics.AttachPrometheusMiddleware(stack, metrics.S3RequestDuration, metrics.S3RequestsTotal)
			},
		}),
	)
	if err != nil {
		return nil, err
	}
	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(params.Endpoint)
	})

	return &S3Client{
		S3Service: s3Client,
	}, nil
}

func (client *S3Client) CreateBucket(ctx context.Context, bucketName string, params util.StorageClientParameters) error {

	input := &s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	if params.Region != util.DefaultRegion {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(params.Region),
		}
	}

	_, err := client.S3Service.CreateBucket(ctx, input)
	return err
}

func (client *S3Client) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := client.S3Service.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &bucketName,
	})
	return err
}
