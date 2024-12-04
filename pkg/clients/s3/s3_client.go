package s3client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
	"k8s.io/klog/v2"
)

type S3API interface {
	CreateBucket(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

const (
	defaultRegion  = "us-east-1"
	requestTimeout = 15 * time.Second
)

type S3Params struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Region    string
	TLSCert   []byte // Optional field for TLS certificates
	Debug     bool
}

type S3Client struct {
	S3Service S3API
}

func InitS3Client(params S3Params) (*S3Client, error) {
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

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(params.AccessKey, params.SecretKey, "")),
		config.WithHTTPClient(httpClient),
		config.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(params.Endpoint)
	})

	return &S3Client{
		S3Service: s3Client,
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

func (client *S3Client) CreateBucket(ctx context.Context, bucketName string, params S3Params) error {

	input := &s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	if params.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(params.Region),
		}
	}

	_, err := client.S3Service.CreateBucket(ctx, input)
	if err != nil {
		return err
	}

	klog.InfoS("Bucket creation operation succeeded", "name", bucketName, "region", params.Region)
	return nil
}
