package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// --- Server Implementation ---

type Server struct {
	Addr     string
	CertFile string
	KeyFile  string
	// CaFile           string // No longer needed
	KnownClientsFile string

	httpServer *http.Server
}

// NewServer creates a new server instance.
func NewServer(addr, certFile, keyFile, knownClientsFile string) *Server {
	return &Server{
		Addr:     addr,
		CertFile: certFile,
		KeyFile:  keyFile,
		// CaFile:           caFile, // Removed
		KnownClientsFile: knownClientsFile,
	}
}

// Start initializes and starts the HTTPS server in a goroutine.
func (s *Server) Start() error {
	log.Println("Configuring server TLS for self-signed client verification...")
	tlsConfig, err := createServerTLSConfig(s.KnownClientsFile) // Pass only knownClientsFile
	if err != nil {
		return fmt.Errorf("failed to create server TLS config: %w", err)
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:      s.Addr,
		TLSConfig: tlsConfig,
		Handler:   http.HandlerFunc(helloHandler), // Use the handler defined below
	}

	log.Printf("Starting HTTPS server on %s...", s.Addr)
	log.Printf("Server expects client CN and Fingerprint to match entries in %s", s.KnownClientsFile)

	// Start server in a goroutine so it doesn't block
	go func() {
		err := s.httpServer.ListenAndServeTLS(s.CertFile, s.KeyFile)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Server ListenAndServeTLS error: %v", err) // Use log.Printf, not Fatalf in goroutine
		} else {
			log.Println("Server stopped gracefully.")
		}
	}()
	// TODO: Add a readiness check mechanism (e.g., channel) if needed before client connects in tests
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return errors.New("server not started")
	}
	log.Println("Stopping server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Add timeout
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// --- Server Handlers & Helpers (belong conceptually with the server) ---

// helloHandler responds to requests.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	cn := "unknown"
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cn = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	log.Printf("Received request from %s for %s", cn, r.URL.Path)
	fmt.Fprintf(w, "Hello, authenticated client '%s'!\n", cn)
}

// loadKnownClients reads the known clients file and parses it.
func loadKnownClients(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open known clients file %s: %w", filePath, err)
	}
	defer file.Close()

	clients := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") { // Skip empty lines and comments
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			log.Printf("Skipping invalid line %d in %s: format should be '<common_name> <fingerprint>'", lineNumber, filePath)
			continue
		}
		cn := strings.TrimSpace(parts[0])
		fingerprint := strings.ToUpper(strings.TrimSpace(parts[1])) // Normalize fingerprint
		if cn == "" || fingerprint == "" {
			log.Printf("Skipping invalid line %d in %s: empty common name or fingerprint", lineNumber, filePath)
			continue
		}
		clients[cn] = fingerprint
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading known clients file %s: %w", filePath, err)
	}

	if len(clients) == 0 {
		log.Printf("Warning: No valid client entries found in %s", filePath)
	}

	return clients, nil
}

// verifyClientCertificate checks if the client certificate matches a known client.
// NOTE: verifiedChains will be nil in the self-signed setup as ClientCAs is not set.
func verifyClientCertificate(rawCerts [][]byte, _ [][]*x509.Certificate, knownClients map[string]string) error {
	if len(rawCerts) == 0 {
		return errors.New("no client certificate provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse client certificate: %w", err)
	}

	hash := sha256.Sum256(cert.Raw)
	var buf strings.Builder
	for i, b := range hash {
		fmt.Fprintf(&buf, "%02X", b)
		if i < len(hash)-1 {
			buf.WriteByte(':')
		}
	}
	fingerprint := buf.String()
	cn := cert.Subject.CommonName

	log.Printf("Verifying client: CN='%s', Fingerprint='%s'", cn, fingerprint)

	knownFingerprint, ok := knownClients[cn]
	if !ok {
		log.Printf("Authentication failed: Client CN '%s' not found in known clients file.", cn)
		return fmt.Errorf("client CN '%s' not authorized", cn)
	}

	if knownFingerprint != fingerprint {
		log.Printf("Authentication failed: Fingerprint mismatch for CN '%s'. Expected '%s', Got '%s'", cn, knownFingerprint, fingerprint)
		return fmt.Errorf("client fingerprint mismatch for CN '%s'", cn)
	}

	log.Printf("Client authenticated successfully via fingerprint: CN='%s'", cn)
	return nil
}
