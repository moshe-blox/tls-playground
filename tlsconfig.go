package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
)

// loadCertPool loads certificates from a PEM file into a cert pool.
// This is still useful for loading the server's cert into the client's trust pool.
func loadCertPool(certFile string) (*x509.CertPool, error) {
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate PEM %s: %w", certFile, err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("failed to append certificate from %s to pool", certFile)
	}
	return certPool, nil
}

// createServerTLSConfig creates a tls.Config for the server.
// It requires client certificates but performs verification *only* via VerifyPeerCertificate.
func createServerTLSConfig(knownClientsFile string) (*tls.Config, error) {
	knownClients, err := loadKnownClients(knownClientsFile)
	if err != nil {
		return nil, fmt.Errorf("error loading known clients from %s: %w", knownClientsFile, err)
	}
	log.Printf("Loaded %d known clients for verification.", len(knownClients))

	// No CA pool for client verification needed here, rely on VerifyPeerCertificate
	cfg := &tls.Config{
		ClientAuth: tls.RequireAnyClientCert, // Require a cert, but don't verify against CAs
		// ClientCAs: nil, // No CA pool specified
		MinVersion: tls.VersionTLS12,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// NOTE: verifiedChains will be nil because we didn't provide ClientCAs.
			// We rely *entirely* on our custom verification logic based on the raw cert.
			if len(rawCerts) == 0 {
				return errors.New("no client certificate presented") // Should be caught by RequireAnyClientCert
			}
			// Perform verification based on fingerprint and CN in knownClients map
			return verifyClientCertificate(rawCerts, nil, knownClients) // Pass nil for verifiedChains
		},
	}

	return cfg, nil
}

// createClientTLSConfig creates a tls.Config for the client.
// It uses the client's cert/key and explicitly trusts the server's certificate.
func createClientTLSConfig(serverCertFile, clientCertFile, clientKeyFile string) (*tls.Config, error) {
	// Load client cert/key for client's identity
	cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client key pair (%s, %s): %w", clientCertFile, clientKeyFile, err)
	}

	// Load server's cert into the RootCAs pool for explicit trust
	rootCAPool, err := loadCertPool(serverCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate %s for client trust: %w", serverCertFile, err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert}, // Client's identity
		RootCAs:      rootCAPool,              // Explicitly trust only certs in this pool (server.crt)
		MinVersion:   tls.VersionTLS12,
		// ServerName check still happens against the CN/SAN in the trusted server.crt
	}

	return cfg, nil
}
