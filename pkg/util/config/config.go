package config

type StorageClientParameters struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Region    string
	TLSCert   []byte // Optional field for TLS certificates
	Debug     bool
}
