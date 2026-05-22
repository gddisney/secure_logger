
# Secure-Logger: Zero-Trust Telemetry & Search Stack

**Secure-Logger** is a high-performance, embedded logging ingest and full-text search engine. It acts as a lightweight, zero-trust alternative to the traditional ELK (Elasticsearch, Logstash, Kibana) stack.

Instead of running multiple heavy Java virtual machines, this stack compiles down to a single binary using a suite of custom, decoupled Go micro-frameworks to handle encrypted ingestion, multi-version storage, Okapi BM25 full-text indexing, and a reactive Live-Component web dashboard.

## 🏗️ Architecture

This project is orchestrated using the following core components:

* **Ingress Edge (`secure_network`)**: Replaces *Logstash*. A zero-trust dual-stack HTTP/3 router that exposes a highly concurrent `/ingest` endpoint over TLS.
* **Search Engine (`orchid_sync`)**: Replaces *Elasticsearch*. An NLP-driven search brain that tokenizes incoming logs and maps them into an inverted B+ Tree index using the Okapi BM25 ranking algorithm.
* **Storage Pool (`ultimate_db`)**: The underlying data node. Stores raw JSON log payloads using Middle-Out compression and handles Optimistic Concurrency Control (OCC) for the search indices.
* **Dashboard (`guikit`)**: Replaces *Kibana*. A reactive UI engine serving GML templates to render a real-time, searchable log viewer.
* **Authentication (`webauthnext`)**: Secures the underlying mesh and provides a Passkey/OIDC identity provider.

## ✨ Features

* **Single-Binary Deployment:** No external dependencies, no JVM, and no separate database instances to manage.
* **Full-Text Search:** Instantly query historical logs using Boolean logic (e.g., `error AND database`).
* **Zero-Allocation Ingestion:** Handles massive log throughput by dropping locks aggressively and utilizing reusable internal scratch buffers.
* **Automated Data Retention:** Logs are written to the transactional database with a 30-day TTL (720 hours); stale logs are swept and vacuumed automatically.
* **CLI Forwarder:** Includes a native UNIX pipe client (`log-forwarder`) to easily tail existing log files directly into the secure mesh.

---

## 🚀 Getting Started

### Prerequisites

* **Go 1.25+** (Required by GUIKit)

### 1. Installation

Clone the repository and pull down the required dependencies:

```bash
git clone https://github.com/your-username/secure-logger.git
cd secure-logger
go mod tidy

```

### 2. Build the Server

Compile and run the primary backend server. This will automatically generate the required `.db` and `.wal` storage files, provision ephemeral TLS certificates, and boot the web dashboard.

```bash
go build -o secure-logger main.go
./secure-logger

```

* **Web Dashboard:** `http://localhost:8080` (or the port defined by your GUIKit run context)
* **Ingest Router:** `https://localhost:443/ingest` (Note: Uses ephemeral self-signed certs)

### 3. Build the CLI Ingest Client

Compile the lightweight CLI forwarder used to pipe terminal output into the server:

```bash
go build -o log-forwarder log_client.go

```

---

## 💻 Usage: Ingesting Logs

The `log-forwarder` reads continuously from standard input (`stdin`) and fires JSON payloads to your ingest endpoint. It automatically parses the text to determine the log severity (INFO, WARN, ERROR) based on keywords.

**Tail a live file (e.g., Nginx, Syslog):**

```bash
tail -f /var/log/nginx/access.log | ./log-forwarder --service="nginx-proxy"

```

**Pipe the output of a running application:**

```bash
node my_app.js | ./log-forwarder --service="node-backend"

```

**Manually test the search engine:**

```bash
echo "Failed to connect to postgres database at 10.0.0.1" | ./log-forwarder --service="db-worker"

```

Once ingested, open the web dashboard in your browser. You can immediately search for `"postgres AND Failed"` to retrieve the log.

---

## 📂 Project Structure

```text
secure-logger/
├── main.go             # The core server (Storage, Search, Auth, and Router)
├── log_client.go       # The CLI pipe forwarder
├── go.mod              # Dependency graph
└── views/
    ├── layout.gml      # The global dashboard shell
    └── index.gml       # The interactive log-search template

```
