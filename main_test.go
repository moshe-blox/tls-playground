package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestIntegrationClientServer performs an integration test of the client and server.
// It assumes ./setup.sh has been run successfully beforehand to generate self-signed certificates.
func TestIntegrationClientServer(t *testing.T) {
	// --- Test Configuration ---
	serverAddr := "localhost:8444" // Use a different port for testing
	serverURL := fmt.Sprintf("https://%s/hello", serverAddr)
	certDir := "certs"
	// caFile := certDir + "/ca.crt" // No longer used
	serverCertFile := certDir + "/server.crt"
	serverKeyFile := certDir + "/server.key"
	clientCertFile := certDir + "/client.crt"
	clientKeyFile := certDir + "/client.key"
	knownClientsFile := certDir + "/knownClients.txt"
	expectedClientCN := "my_secure_client" // From setup.sh CLIENT_SUBJ

	// --- Server Setup ---
	t.Logf("Starting server on %s", serverAddr)
	// Call NewServer without caFile
	server := NewServer(serverAddr, serverCertFile, serverKeyFile, knownClientsFile)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	// Make sure server is stopped even if test fails
	defer func() {
		t.Log("Stopping server...")
		if err := server.Stop(); err != nil {
			t.Errorf("Failed to stop server cleanly: %v", err)
		}
	}()

	// Crude way to wait for server start. In production, use health checks or channels.
	t.Log("Waiting for server to start...")
	time.Sleep(1 * time.Second)

	// --- Client Setup ---
	t.Logf("Creating client for %s", serverURL)
	// Call NewClient with serverCertFile instead of caFile
	client, err := NewClient(serverURL, serverCertFile, clientCertFile, clientKeyFile)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// --- Send Request & Assert ---
	t.Log("Sending request...")
	body, statusCode, err := client.SendRequest()
	if err != nil {
		t.Fatalf("Client request failed: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	expectedBodyPart := fmt.Sprintf("Hello, authenticated client '%s'!", expectedClientCN)
	if !strings.Contains(body, expectedBodyPart) {
		t.Errorf("Expected response body to contain '%s', got '%s'", expectedBodyPart, body)
	}

	t.Log("Integration test successful!")
	// Server Stop is handled by defer
}
