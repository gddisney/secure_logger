package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gddisney/guikit"
	"github.com/gddisney/orchid_sync"
	"github.com/gddisney/secure_network"
	"github.com/gddisney/ultimate_db"
	"github.com/gddisney/webauthnext"
)

// LogPayload represents the incoming JSON telemetry
type LogPayload struct {
	Level   string `json:"level"`
	Service string `json:"service"`
	Message string `json:"message"`
}

// LogDisplay represents the parsed data sent to index.gml for rendering
type LogDisplay struct {
	LevelClass string
	Level      string
	Time       string
	Service    string
	Message    string
}

// generateID creates a quick unique identifier for our logs
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func main() {
	// ==========================================
	// 1. STORAGE: ultimate_db Initialization
	// ==========================================
	disk, err := ultimate_db.NewDiskManager("logs.db")
	if err != nil {
		log.Fatalf("Failed to open logs.db: %v", err)
	}
	pool := ultimate_db.NewBufferPool(disk, 2048)
	wal, err := ultimate_db.NewBatchingWAL("logs.wal")
	if err != nil {
		log.Fatalf("Failed to open logs.wal: %v", err)
	}
	db := ultimate_db.NewDB(pool, wal)
	defer db.Close()
	
	logPageID := ultimate_db.PageID(1)

	// ==========================================
	// 2. DASHBOARD: guikit Initialization
	// ==========================================
	ui, err := guikit.New("ui.db", "ui.wal")
	if err != nil {
		log.Fatalf("Failed to boot guikit: %v", err)
	}

	// ==========================================
	// 3. AUTH: webauthnext Initialization
	// ==========================================
	authProvider, err := webauthnext.New(ui, "LogOps Console", "localhost", "https://localhost:443")
	if err != nil {
		log.Fatalf("Failed to boot webauthnext provider: %v", err)
	}

	// ==========================================
	// 4. SEARCH: orchid_sync Initialization
	// ==========================================
	searchEngine, err := orchid_sync.NewEngine("logs_index.db", 9999, authProvider)
	if err != nil {
		log.Fatalf("Failed to boot search engine: %v", err)
	}

	// Setup the main dashboard route
	ui.Get("/", func(c *guikit.Context) {
		// FIXED: Using c.R as defined in guikit/engine.go
		var query string
		if c.R != nil {
			query = c.R.URL.Query().Get("q")
		}

		var searchResults []LogDisplay

		if query != "" {
			hits, err := searchEngine.Search(query, 50)
			if err == nil {
				readTxn := db.BeginTxn()
				
				for _, hit := range hits {
					// NOTE: Assuming 'DocID' is the string identifier in orchid_sync.SearchResult.
					// If it is named differently (e.g., 'Key' or 'DocumentID'), change it here.
					rawLog, err := db.ReadCompressed(logPageID, readTxn, []byte(hit.DocID))
					if err != nil {
						continue 
					}

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
							Time:       time.Now().Format("2006-01-02 15:04:05"), 
							Service:    parsedLog.Service,
							Message:    parsedLog.Message,
						})
					}
				}
			}
		}

		c.Data["Query"] = query
		c.Data["Results"] = searchResults
		ui.Render(c, "views/index") 
	})

	// ==========================================
	// 5. INGRESS: secure_network Initialization
	// ==========================================
	r, err := secure_network.NewRouter(db, ui, "secure_session_token")
	if err != nil {
		log.Fatalf("Kernel initialization failed: %v", err)
	}
	r.Port = "443"

	r.Mux.HandleFunc("/ingest", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		logID := generateID()

		txnID := db.BeginTxn()
		err = db.WriteCompressed(logPageID, txnID, []byte(logID), payload, 720*time.Hour)
		if err != nil {
			log.Printf("Failed to write log to DB: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var logData LogPayload
		if err := json.Unmarshal(payload, &logData); err == nil {
			indexableText := logData.Level + " " + logData.Service + " " + logData.Message
			searchEngine.Index(logID, indexableText)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Log ingested successfully"))
	})

	log.Println("Booting Zero-Trust Ingress Router on :443")
	r.Boot()
}
