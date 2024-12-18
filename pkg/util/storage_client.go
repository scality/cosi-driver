package util

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	c "github.com/scality/cosi-driver/pkg/constants"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
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
	IAMEndpoint     string // Optional field for IAM endpoint(default: Endpoint)
	Region          string // Optional field for region
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

func ConfigureTLSTransport(certData []byte) *http.Transport {
	tlsSettings := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if len(certData) > 0 {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certData); !ok {
			klog.Warning("Failed to append provided cert data to the certificate pool")
		}
		tlsSettings.RootCAs = caCertPool
	} else {
		klog.V(c.LvlDebug).Info("No certificate data provided; skipping TLS verification")
		tlsSettings.InsecureSkipVerify = true
	}

	return &http.Transport{
		TLSClientConfig: tlsSettings,
	}
}
