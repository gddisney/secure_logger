package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gddisney/secure_network"
	"github.com/gddisney/ultimate_db"
)

// LogPayload must match the gateway's expected ingestion format
type LogPayload struct {
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

func main() {
	// 1. Setup CLI flags
	gatewayAddr := flag.String("gateway", "gateway.mesh.internal:443", "Gateway QUIC address")
	serviceName := flag.String("service", "slog-pipe", "Service name")
	flag.Parse()

	// 2. Initialize a local "headless" EdgeNode components
	// We need a DB for identity storage, but we can use an in-memory disk manager
	// or a temporary file since this is a transient logging client.
	dm, _ := ultimate_db.NewDiskManager("client_identity.db")
	bp := ultimate_db.NewBufferPool(dm, 64)
	wal, _ := ultimate_db.NewBatchingWAL("client_identity.wal")
	db := ultimate_db.NewDB(bp, wal)
	defer db.Close()

	// 3. Instantiate MeshNode (this handles the Noise Handshake + DBSC)
	// You need to ensure the Gateway public key is known to the client
	gatewayPubKey := []byte("central-gateway-static-pubkey-32b") 
	meshNode, err := secure_network.NewMeshNode(db, gatewayPubKey)
	if err != nil {
		log.Fatalf("Failed to init mesh node: %v", err)
	}

	// 4. Connect to the mesh gateway
	fmt.Printf("📡 Establishing secure mesh tunnel to %s...\n", *gatewayAddr)
	if err := meshNode.Connect(*gatewayAddr); err != nil {
		log.Fatalf("Mesh connection failed: %v", err)
	}
	fmt.Println("✅ Secure overlay connected.")

	// 5. Pipe loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Wrap the log in a payload
		payload := LogPayload{
			Level:   "INFO",
			Service: *serviceName,
			Message: line,
		}
		
		// Encode to JSON
		payloadBytes, _ := json.Marshal(payload)

		// 6. Send via Mesh RPC (Action: "rpc", Content: JSON RPC)
		// This routes directly to your Gateway's routeToAPI() logic
		rpcArgs := map[string]interface{}{
			"method": "ingest_log",
			"args":   payloadBytes,
		}
		argsBytes, _ := json.Marshal(rpcArgs)
		
		err := meshNode.SendAction(secure_network.APIPayload{
			Action:  "rpc",
			Content: string(argsBytes),
		})

		if err != nil {
			fmt.Printf("❌ Failed to send log: %v\n", err)
		}
	}
}
