package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gddisney/guikit"
	"github.com/gddisney/orchid_sync"
	"github.com/gddisney/secure_bootstrap"
	"github.com/gddisney/secure_network"
	"github.com/gddisney/ultimate_db"
	"github.com/gddisney/webauthnext"
)

type LogPayload struct {
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

type LogDisplay struct {
	LevelClass string
	Level      string
	Time       string
	Service    string
	Message    string
}

var (
	recentLogs []LogDisplay
	logsMu     sync.RWMutex
)

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func recordInternalLog(db *ultimate_db.DB, logPageID ultimate_db.PageID, searchEngine *orchid_sync.Engine, level, service, message string) {
	logID := generateID()
	logData := LogPayload{Level: level, Service: service, Message: message}
	payload, _ := json.Marshal(logData)

	txnID := db.BeginTxn()
	_ = db.WriteCompressed(logPageID, txnID, []byte(logID), payload, 720*time.Hour)
	db.CommitTxn(txnID) // Data now permanently saves to disk

	indexableText := strings.ToLower(level + " " + service + " " + message)
	_ = searchEngine.Index(logID, indexableText)
	
	levelClass := "level-info"
	if level == "ERROR" || level == "FATAL" { levelClass = "level-error" } else if level == "WARN" { levelClass = "level-warn" }

	logsMu.Lock()
	recentLogs = append([]LogDisplay{{
		LevelClass: levelClass,
		Level:      level,
		Time:       time.Now().Format("15:04:05"), 
		Service:    service,
		Message:    message,
	}}, recentLogs...)
	if len(recentLogs) > 100 {
		recentLogs = recentLogs[:100]
	}
	logsMu.Unlock()

	log.Printf("[INTERNAL] 📝 %s %s: %s", level, service, message)
}

// formatBooleanQuery ensures that standard space-separated words are treated as an AND query
// if the user didn't explicitly use the new AST boolean operators.
func formatBooleanQuery(q string) string {
	upper := strings.ToUpper(q)
	if strings.Contains(upper, " AND ") || strings.Contains(upper, " OR ") || strings.Contains(upper, " NOT ") {
		return q 
	}
	
	tokens := strings.Fields(q)
	return strings.Join(tokens, " AND ")
}

func main() {
	// 1. Initialize the UI Framework
	ui, err := guikit.New("ui.db", "ui.wal")
	if err != nil { log.Fatalf("Failed to boot guikit: %v", err) }

	authProvider, err := webauthnext.New(ui, "LogOps Console", "localhost", "https://localhost")
	if err != nil { log.Fatalf("Failed to boot webauthnext: %v", err) }

	// 2. Boot Search Engine (this automatically initializes the DB and secure_network.EdgeNode)
	searchEngine, err := orchid_sync.NewEngine("logs.db", 443, authProvider)
	if err != nil { log.Fatalf("Failed to boot search engine: %v", err) }

	// 3. Extract encapsulated EdgeNode infrastructure
	edgeNode := searchEngine.NetNode()
	db := edgeNode.DB
	r := edgeNode.Router
	logPageID := ultimate_db.PageID(1)

	// Attach UI to the EdgeNode's Router
	r.GUIKit = ui
	r.Mux.Handle("/", ui.Mux)

	// 4. Mesh Initialization for Outbound Tunnel
	gatewayPubKey := []byte("central-gateway-static-pubkey-32b") 
	gatewayAddress := "gateway.mesh.internal:443"

	meshNode, err := secure_network.NewMeshNode(db, gatewayPubKey)
	if err != nil { log.Fatalf("Mesh Node hardware identity instantiation failed: %v", err) }

	// --- BOOTSTRAP IDENTITY & INJECT MESH ---
	secure_bootstrap.BootstrapAuth(r, authProvider, meshNode, gatewayAddress)

	// 5. Replace DB polling loop with Event-Driven RPC Manager
	rpcEngine := r.Modules["mesh_rpc"].(*secure_network.RPCManager)
	rpcEngine.Register("ingest_log", func(ctx secure_network.RPCContext, args []byte) (interface{}, error) {
		var logData LogPayload
		if err := json.Unmarshal(args, &logData); err != nil {
			return nil, err
		}
		
		recordInternalLog(db, logPageID, searchEngine, logData.Level, logData.Service, logData.Message)
		log.Printf("[RPC] Processed log ingestion from peer: %x", ctx.CallerID[:8])
		
		return map[string]string{"status": "success"}, nil
	})

	// --- SETUP UI ROUTES ---
	ui.Get("/logout", func(c *guikit.Context) {
		http.SetCookie(c.W, &http.Cookie{Name: "session_id", MaxAge: -1, Path: "/"})
		http.Redirect(c.W, c.R, "/auth", http.StatusSeeOther)
	})

	ui.Get("/", secure_bootstrap.RequireAuth(r, func(c *guikit.Context) {
		var query string
		var searchError string

		if c.R != nil {
			query = strings.TrimSpace(c.R.URL.Query().Get("q"))
		}

		var searchResults []LogDisplay

		if query != "" {
			formattedQuery := formatBooleanQuery(query)
			hits, err := searchEngine.Search(formattedQuery, 50)
			
			if err != nil {
				searchError = "Invalid query syntax: " + err.Error()
			} else {
				readTxn := db.BeginTxn()
				for _, hit := range hits {
					rawLog, err := db.ReadCompressed(logPageID, readTxn, []byte(hit.DocID))
					if err != nil { continue }

					var parsedLog LogPayload
					if err := json.Unmarshal(rawLog, &parsedLog); err == nil {
						levelClass := "level-info"
						if parsedLog.Level == "ERROR" || parsedLog.Level == "FATAL" { 
							levelClass = "level-error" 
						} else if parsedLog.Level == "WARN" { 
							levelClass = "level-warn" 
						}

						searchResults = append(searchResults, LogDisplay{
							LevelClass: levelClass,
							Level:      parsedLog.Level,
							Time:       time.Now().Format("15:04:05"), 
							Service:    parsedLog.Service,
							Message:    parsedLog.Message,
						})
					}
				}
				db.CommitTxn(readTxn)
			}
		} else {
			logsMu.RLock()
			searchResults = append(searchResults, recentLogs...)
			logsMu.RUnlock()
		}

		c.Data["Query"] = query
		c.Data["Results"] = searchResults
		c.Data["SearchError"] = searchError 
		ui.Render(c, "views/index") 
	}))

	r.Mux.HandleFunc("/ingest", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost { return }
		payload, err := io.ReadAll(req.Body)
		if err != nil { return }
		defer req.Body.Close()

		var logData LogPayload
		if err := json.Unmarshal(payload, &logData); err == nil {
			recordInternalLog(db, logPageID, searchEngine, logData.Level, logData.Service, logData.Message)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Log ingested successfully"))
	})

	log.Println("Booting Zero-Trust Ingress Edge Node on :443")
	// Launch the unified Edge Node (which triggers the kernel Boot protocol and listeners)
	if err := edgeNode.Start("443", r.TLSConfig); err != nil {
		log.Fatalf("Edge Node crashed: %v", err)
	}
}
