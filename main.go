package main

import (
	"fmt"
	"log"

	// Ensure you have run 'go mod tidy' or 'go get github.com/alecthomas/kong'
	"github.com/alecthomas/kong"
)

// --- CLI Structure ---

// ServerCmd defines the kong command for the server.
type ServerCmd struct {
	CertFile     string `kong:"name='cert',help='Server certificate file.',default='certs/server.crt',type='path'"`
	KeyFile      string `kong:"name='key',help='Server private key file.',default='certs/server.key',type='path'"`
	KnownClients string `kong:"name='known-clients',help='File listing authorized client CNs and fingerprints.',default='certs/knownClients.txt',type='path'"`
	Addr         string `kong:"name='addr',help='Address to listen on.',default=':8443'"`
}

// Run starts the server using the Server struct from server.go.
func (s *ServerCmd) Run() error {
	server := NewServer(s.Addr, s.CertFile, s.KeyFile, s.KnownClients)
	err := server.Start() // Start runs the server in a goroutine
	if err != nil {
		// Use log.Fatalf only in main or test setup, return error here
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Keep the main goroutine alive. Server runs in its own goroutine.
	// In a real app, you might wait on a signal channel here for graceful shutdown.
	log.Println("Server started. Running indefinitely...")
	select {}
}

// ClientCmd defines the kong command for the client.
type ClientCmd struct {
	CertFile       string `kong:"name='cert',help='Client certificate file.',default='certs/client.crt',type='path'"`
	KeyFile        string `kong:"name='key',help='Client private key file.',default='certs/client.key',type='path'"`
	ServerCertFile string `kong:"name='server-cert',help='Server certificate file for client verification.',default='certs/server.crt',type='path'"`
	ServerURL      string `kong:"name='url',help='Server URL to connect to.',default='https://localhost:8443/hello'"`
}

// Run executes the client request using the Client struct from client.go.
func (c *ClientCmd) Run() error {
	client, err := NewClient(c.ServerURL, c.ServerCertFile, c.CertFile, c.KeyFile)
	if err != nil {
		// Use log.Fatalf only in main or test setup, return error here
		return fmt.Errorf("failed to create client: %w", err)
	}

	_, _, err = client.SendRequest()
	if err != nil {
		return fmt.Errorf("client request failed: %w", err)
	}
	// Response is printed within SendRequest for interactive use
	return nil
}

// --- Main CLI Definition & Execution ---

var cli struct {
	Server ServerCmd `kong:"cmd,help='Run the mTLS server with known client verification.'"`
	Client ClientCmd `kong:"cmd,help='Run the mTLS client.'"`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("tls-playground"),
		kong.Description("A playground CLI for mTLS with known client/server verification."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	// kong.Parse returns the parsed command context (ctx)
	// ctx.Run() executes the Run() method of the selected command (ServerCmd or ClientCmd)
	err := ctx.Run()
	// ctx.FatalIfErrorf handles the error returned from the command's Run method
	// It prints the error and exits with a non-zero status if err is not nil.
	ctx.FatalIfErrorf(err)
}
