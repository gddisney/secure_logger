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
	targetURL := flag.String("url", "https://localhost:443/ingest", "The secure-logger ingest URL")
	serviceName := flag.String("service", "slog-pipe", "The service name to tag these logs with")
	flag.Parse()

	// Configure HTTP client to ignore self-signed certificates on local ingestion
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("📡 Listening for slog JSON on stdin and forwarding to %s...\n", *targetURL)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var payload LogPayload
		var rawJSON map[string]interface{}

		// 1. Attempt to parse the line as a JSON object (slog default behavior)
		if err := json.Unmarshal([]byte(line), &rawJSON); err == nil {
			
			// Extract standard slog fields (they default to lowercase keys)
			level := "INFO"
			if lvl, ok := rawJSON["level"].(string); ok {
				level = strings.ToUpper(lvl)
			}

			msg := ""
			if m, ok := rawJSON["msg"].(string); ok {
				msg = m
			} else if m, ok := rawJSON["message"].(string); ok {
				msg = m
			}

			// Clean up extracted keys so we can format the remaining custom slog attributes
			delete(rawJSON, "level")
			delete(rawJSON, "msg")
			delete(rawJSON, "message")
			delete(rawJSON, "time") // Optional: remove timestamp if your dashboard tracks receipt time

			// Format remaining attributes (like custom fields added via slog.String("user", "admin"))
			extraData, _ := json.Marshal(rawJSON)
			if string(extraData) != "{}" {
				msg = fmt.Sprintf("%s | Attributes: %s", msg, string(extraData))
			}

			payload = LogPayload{
				Level:   level,
				Service: *serviceName,
				Message: msg,
			}

		} else {
			// 2. Fallback: Not JSON, parse as raw terminal text
			level := "INFO"
			upperLine := strings.ToUpper(line)
			if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "FAIL") || strings.Contains(upperLine, "PANIC") {
				level = "ERROR"
			} else if strings.Contains(upperLine, "WARN") {
				level = "WARN"
			}

			payload = LogPayload{
				Level:   level,
				Service: *serviceName,
				Message: line,
			}
		}

		// Ship to secure_network
		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to encode JSON: %v\n", err)
			continue
		}

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
