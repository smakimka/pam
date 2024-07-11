package certs

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"

	"google.golang.org/grpc/credentials"
)

//go:embed ca-cert.pem
var pemServerCA []byte

func LoadTLSCredentials() (credentials.TransportCredentials, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}
