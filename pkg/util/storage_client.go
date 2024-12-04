package util

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Constants for storage client configuration
const (
	DefaultRegion         = "us-east-1"
	DefaultRequestTimeout = 15 * time.Second
)

// StorageClientParameters holds configuration for S3/IAM clients.
type StorageClientParameters struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Region          string
	TLSCert         []byte // Optional field for TLS certificates
	Debug           bool   // Optional field for debug mode
}

// NewStorageClientParameters initializes default storage client parameters.
func NewStorageClientParameters() *StorageClientParameters {
	return &StorageClientParameters{
		Region: DefaultRegion,
		Debug:  false,
	}
}

// Validate checks that all required fields are set.
func (p *StorageClientParameters) Validate() error {
	if p.AccessKeyID == "" {
		return status.Error(codes.InvalidArgument, "accessKeyID is required")
	}
	if p.SecretAccessKey == "" {
		return status.Error(codes.InvalidArgument, "secretAccessKey is required")
	}
	if p.Endpoint == "" {
		return status.Error(codes.InvalidArgument, "endpoint is required")
	}
	return nil
}
