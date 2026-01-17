package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/emulator"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/testutil"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to listen on")
	apiVersion := flag.String("api-version", "58.0", "Salesforce API version to emulate")
	clientID := flag.String("client-id", "test_client_id", "OAuth client ID")
	clientSecret := flag.String("client-secret", "test_client_secret", "OAuth client secret")
	username := flag.String("username", "test@example.com", "OAuth username")
	password := flag.String("password", "testpassword", "OAuth password")
	loadFixtures := flag.String("load-fixtures", "", "Load a pre-built fixture scenario (empty_org, basic_crm, high_volume)")
	flag.Parse()

	// Create emulator with options
	emu := emulator.New(
		emulator.WithAPIVersion(*apiVersion),
		emulator.WithCredentials(auth.Credential{
			ClientID:     *clientID,
			ClientSecret: *clientSecret,
			Username:     *username,
			Password:     *password,
		}),
		emulator.WithPort(*port),
	)

	// Start the emulator
	baseURL := emu.Start()
	defer emu.Stop()

	// Load fixtures if requested
	if *loadFixtures != "" {
		if err := testutil.LoadScenario(emu, *loadFixtures); err != nil {
			log.Fatalf("Failed to load fixtures: %v", err)
		}
		fmt.Printf("Loaded fixture scenario: %s\n", *loadFixtures)
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          Salesforce Emulator - go-salesforce-emulator          ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Base URL:      %-47s ║\n", baseURL)
	fmt.Printf("║  API Version:   %-47s ║\n", *apiVersion)
	fmt.Printf("║  Client ID:     %-47s ║\n", *clientID)
	fmt.Printf("║  Username:      %-47s ║\n", *username)
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Endpoints:                                                    ║")
	fmt.Printf("║    OAuth:     %s/services/oauth2/token\n", baseURL)
	fmt.Printf("║    REST API:  %s/services/data/v%s/\n", baseURL, *apiVersion)
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Press Ctrl+C to stop the emulator                             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Also set up a simple HTTP server on the specified port
	// The test server uses a random port, so we need a separate server for the CLI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Proxy to the test server
		client := emu.HTTPClient()
		req, _ := http.NewRequest(r.Method, baseURL+r.URL.Path+"?"+r.URL.RawQuery, r.Body)
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)

		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				_, _ = w.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	})

	go func() {
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("Starting proxy server on %s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-sigChan
	fmt.Println("\nShutting down emulator...")
}
