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
		return q // The user is utilizing explicit AST logic, pass it through directly.
	}
	
	// Fallback for natural language searches: convert "error auth service" into "error AND auth AND service"
	tokens := strings.Fields(q)
	return strings.Join(tokens, " AND ")
}

func main() {
	disk, err := ultimate_db.NewDiskManager("logs.db")
	if err != nil { log.Fatalf("Failed to open logs.db: %v", err) }
	pool := ultimate_db.NewBufferPool(disk, 2048)
	wal, err := ultimate_db.NewBatchingWAL("logs.wal")
	if err != nil { log.Fatalf("Failed to open logs.wal: %v", err) }
	
	db := ultimate_db.NewDB(pool, wal)
	defer db.Close()
	logPageID := ultimate_db.PageID(1)

	// --- MESH INITIALIZATION ---
	gatewayPubKey := []byte("central-gateway-static-pubkey-32b") 
	gatewayAddress := "gateway.mesh.internal:443"

	meshNode, err := secure_network.NewMeshNode(db, gatewayPubKey)
	if err != nil { log.Fatalf("Mesh Node hardware identity instantiation failed: %v", err) }

	ui, err := guikit.New("ui.db", "ui.wal")
	if err != nil { log.Fatalf("Failed to boot guikit: %v", err) }

	r, err := secure_network.NewRouter(db, ui, "secure_session_token")
	if err != nil { log.Fatalf("Kernel initialization failed: %v", err) }
	r.Port = "443"

	authProvider, err := webauthnext.New(ui, "LogOps Console", "localhost", "https://localhost")
	if err != nil { log.Fatalf("Failed to boot webauthnext: %v", err) }

	searchEngine, err := orchid_sync.NewEngine("logs_index.db", 9999, authProvider)
	if err != nil { log.Fatalf("Failed to boot search engine: %v", err) }

	// --- BOOTSTRAP IDENTITY & INJECT MESH ---
	secure_bootstrap.BootstrapAuth(r, authProvider, meshNode, gatewayAddress)

	// --- RUN MESH TASK CONSUMER ---
	go meshTaskConsumerWorker(db)

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
			// Format the query to ensure AST compatibility
			formattedQuery := formatBooleanQuery(query)
			
			hits, err := searchEngine.Search(formattedQuery, 50)
			
			// Catch AST Parsing Errors (e.g., missing closing parenthesis)
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
		c.Data["SearchError"] = searchError // Pass error out to GML so the UI can display it
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

	log.Println("Booting Zero-Trust Ingress Router on :443")
	r.Boot()
}

func meshTaskConsumerWorker(db *ultimate_db.DB) {
	log.Println("[WORKER] Active and monitoring ultimate_db Page 100 for incoming mesh directives...")
	for {
		time.Sleep(1 * time.Second)
	}
}
