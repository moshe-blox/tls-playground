package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// --- Client Implementation ---

type Client struct {
	ServerURL string
	CertFile  string
	KeyFile   string
	// CaFile    string // No longer needed, trust server cert directly

	httpClient *http.Client
}

// NewClient creates a new client instance.
// It trusts the specific server certificate provided in serverCertFile.
func NewClient(serverURL, serverCertFile, clientCertFile, clientKeyFile string) (*Client, error) {
	tlsConfig, err := createClientTLSConfig(serverCertFile, clientCertFile, clientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create client TLS config: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		ServerURL: serverURL,
		CertFile:  clientCertFile,
		KeyFile:   clientKeyFile,
		// CaFile:     caFile, // Removed
		httpClient: httpClient,
	}, nil
}

// SendRequest sends a GET request to the configured server URL.
func (c *Client) SendRequest() (string, int, error) {
	log.Printf("Sending request to %s...", c.ServerURL)
	resp, err := c.httpClient.Get(c.ServerURL)
	if err != nil {
		// Don't log fatal here, return the error for the caller (e.g., test) to handle
		return "", 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response: Status Code %d", resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	body := string(bodyBytes)
	fmt.Printf("Server Response:\n%s", body) // Keep log for interactive use

	return body, resp.StatusCode, nil
}
