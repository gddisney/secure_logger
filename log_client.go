package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/gddisney/secure_network"
	"github.com/gddisney/ultimate_db"
)

// SignedLogPayload enforces the attribution requirement.
type SignedLogPayload struct {
	Payload   LogPayload `json:"payload"`
	SignerKey string     `json:"signer_key"`
	Signature string     `json:"signature"`
}

type LogPayload struct {
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

func main() {
	gatewayAddr := flag.String("gateway", "localhost:9000", "Gateway QUIC address")
	serviceName := flag.String("service", "slog-pipe", "Service name")
	pubKeyHex := flag.String("pubkey", "", "Gateway Noise Public Key (hex)")
	flag.Parse()

	if *pubKeyHex == "" {
		log.Fatal("? You must provide the gateway's 32-byte public key via -pubkey")
	}

	gatePub, err := hex.DecodeString(*pubKeyHex)
	if err != nil || len(gatePub) != 32 {
		log.Fatalf("? Invalid public key format: %v", err)
	}

	// 1. Initialize local identity storage (Transient or Persisted)
	dm, _ := ultimate_db.NewDiskManager("client_identity.db")
	bp := ultimate_db.NewBufferPool(dm, 64)
	wal, _ := ultimate_db.NewBatchingWAL("client_identity.wal")
	db := ultimate_db.NewDB(bp, wal)
	defer db.Close()

	// 2. Instantiate MeshNode (Handles Noise Handshake + Ed25519 Identity)
	meshNode, err := secure_network.NewMeshNode(db, gatePub)
	if err != nil {
		log.Fatalf("Failed to init mesh node: %v", err)
	}

	// 3. Connect to the mesh
	fmt.Printf("?? Establishing secure mesh tunnel to %s...\n", *gatewayAddr)
	if err := meshNode.Connect(*gatewayAddr); err != nil {
		log.Fatalf("Mesh connection failed: %v", err)
	}
	fmt.Println("? Secure overlay connected.")

	// 4. Pipe loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" { continue }

		// Wrap and Sign Payload
		payload := LogPayload{Level: "INFO", Service: *serviceName, Message: line}
		payloadBytes, _ := json.Marshal(payload)
		
		// Sign with the node's Ed25519 identity key
		// Assuming we retrieve the private key from our MeshNode's storage
		// For implementation: replace with your actual DB retrieval logic
		signature := ed25519.Sign(meshNode.GetDBSCPrivKey(), payloadBytes)
		
		signed := SignedLogPayload{
			Payload:   payload,
			SignerKey: hex.EncodeToString(meshNode.GetNoisePubKey()),
			Signature: base64.StdEncoding.EncodeToString(signature),
		}
		
		signedBytes, _ := json.Marshal(signed)

		// 5. Send via Mesh RPC (The PEP verifies the signature at the gateway)
		rpcArgs := map[string]interface{}{
			"method": "ingest_signed_log",
			"args":   signedBytes,
		}
		argsBytes, _ := json.Marshal(rpcArgs)

		err := meshNode.SendAction(secure_network.APIPayload{
			Action:  "rpc",
			Content: string(argsBytes),
		})

		if err != nil {
			fmt.Printf("? Failed to send log: %v\n", err)
		}
	}
}
