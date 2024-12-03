package types

type StorageClientParameters struct {
	AccessKey   string
	SecretKey   string
	Endpoint    string
	IAMEndpoint string // Optional if different from Endpoint
	Region      string
	TLSCert     []byte // Optional field for TLS certificates
	Debug       bool
}
