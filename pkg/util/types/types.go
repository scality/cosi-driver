package types

type StorageClientParameters struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Region          string
	IAMEndpoint     string // Optional if different from Endpoint
	TLSCert         []byte // Optional field for TLS certificates
	Debug           bool   // Optional field for debug mode for IAM and S3 requests
}
