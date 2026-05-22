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

type LogPayload struct {
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

func main() {
	targetURL := flag.String("url", "https://localhost:443/ingest", "The secure-logger ingest URL")
	serviceName := flag.String("service", "slog-pipe", "The service name to tag these logs with")
	flag.Parse()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}

	// Check if stdin is actually receiving piped data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Println("⚠️  WARNING: No pipe detected. (Did you use the '|' operator?) Type manually and press Enter:")
	} else {
		fmt.Printf("📡 Connected to pipe. Forwarding to %s...\n", *targetURL)
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("\n[DEBUG-1] Read line from stdin: %s\n", line)

		if strings.TrimSpace(line) == "" {
			fmt.Println("[DEBUG-2] Line was empty, skipping.")
			continue
		}

		payload := LogPayload{
			Level:   "INFO",
			Service: *serviceName,
			Message: line,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("❌ [DEBUG-3] JSON Encode failed: %v\n", err)
			continue
		}
		fmt.Printf("[DEBUG-3] Generated JSON Payload: %s\n", string(jsonData))

		fmt.Println("[DEBUG-4] Firing HTTP POST to server...")
		resp, err := client.Post(*targetURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("❌ [DEBUG-5] NETWORK ERROR: %v\n", err)
			continue
		}
		
		fmt.Printf("[DEBUG-5] Server responded with Status: %d %s\n", resp.StatusCode, resp.Status)
		resp.Body.Close()
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("❌ Fatal scanner error: %v\n", err)
	}
	
	fmt.Println("[DEBUG-6] Scanner closed/EOF reached. Exiting.")
}
