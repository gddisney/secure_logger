package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// LogPayload matches the structure expected by your secure-logger backend
type LogPayload struct {
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

func main() {
	// CLI Flags for configuration
	targetURL := flag.String("url", "https://localhost:443/ingest", "The secure-logger ingest URL")
	serviceName := flag.String("service", "cli-pipe", "The service name to tag these logs with")
	flag.Parse()

	// Configure HTTP client to ignore the self-signed/ephemeral certs from secure_network
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second, // Prevent hanging on dead network
	}

	// Read continuously from standard input
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Printf("📡 Listening on stdin and forwarding to %s (Service: %s)...\n", *targetURL, *serviceName)

	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Basic heuristic to detect log severity from the raw text
		level := "INFO"
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "FAIL") || strings.Contains(upperLine, "PANIC") {
			level = "ERROR"
		} else if strings.Contains(upperLine, "WARN") {
			level = "WARN"
		}

		// Construct the JSON payload
		payload := LogPayload{
			Level:   level,
			Service: *serviceName,
			Message: line,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to encode JSON: %v\n", err)
			continue
		}

		// Fire off the HTTP POST to the ingest node
		resp, err := client.Post(*targetURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to send log to %s: %v\n", *targetURL, err)
			continue
		}
		
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "⚠️ Server rejected log, status: %s\n", resp.Status)
		}
		
		resp.Body.Close()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error reading stdin: %v\n", err)
	}
}
