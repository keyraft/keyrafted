# Keyraft

[![GitHub release](https://img.shields.io/github/v/release/keyraft/keyrafted?style=flat-square)](https://github.com/keyraft/keyrafted/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/keyraft/keyrafted?style=flat-square)](https://hub.docker.com/r/keyraft/keyrafted)
[![Go Version](https://img.shields.io/github/go-mod/go-version/keyraft/keyrafted?style=flat-square)](https://go.dev/)
[![License](https://img.shields.io/github/license/keyraft/keyrafted?style=flat-square)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/keyraft/keyrafted/ci.yml?branch=main&style=flat-square)](https://github.com/keyraft/keyrafted/actions)

> A lightweight, self-hosted configuration and secrets management system.

Keyraft stores configuration and secrets securely, manages versioning, and provides live updates through a simple HTTP API.

---

## Features

* ✅ Key-value store with versioning
* ✅ Encrypted storage (AES-256-GCM) for secrets
* ✅ Namespaces for isolation (`project/environment/service`)
* ✅ Token-based authentication with scoped API keys
* ✅ Historical version tracking
* ✅ Watch API for live updates (SSE streaming + long-polling)
* ✅ HTTP/JSON protocol
* ✅ Prometheus metrics endpoint
* ✅ Go SDK with caching and auto-reload

---

## Installation

### Using Docker (Recommended)

```bash
docker run -d -p 7200:7200 \
  -e KEYRAFT_MASTER_KEY=$(openssl rand -base64 32) \
  -v keyraft-data:/data \
  keyraft/keyrafted:latest
```

**Docker Images:**
- **Docker Hub:** [keyraft/keyrafted](https://hub.docker.com/r/keyraft/keyrafted)
- **GitHub Container Registry:** `ghcr.io/keyraft/keyrafted:latest` - [View Packages](https://github.com/keyraft/keyrafted/pkgs/container/keyrafted)

The container automatically initializes on first run. Get the root token from logs:

```bash
# View full logs (token appears after "Root token" message)
docker logs <container-name>
```

### Using Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/keyraft/keyrafted/main/install.sh | sh
```

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/keyraft/keyrafted/releases/latest):

```bash
# Linux
wget https://github.com/keyraft/keyrafted/releases/latest/download/keyrafted-linux-amd64
chmod +x keyrafted-linux-amd64
sudo mv keyrafted-linux-amd64 /usr/local/bin/keyrafted

# macOS
wget https://github.com/keyraft/keyrafted/releases/latest/download/keyrafted-darwin-amd64
chmod +x keyrafted-darwin-amd64
sudo mv keyrafted-darwin-amd64 /usr/local/bin/keyrafted
```

### From Source

```bash
go install github.com/keyraft/keyrafted@latest
```

---

## Quick Start

### 1. Initialize Database

```bash
keyrafted init --data-dir ./data
```

**Output:**
```
✓ Keyraft initialized successfully!

Root token (save this securely):
  eyhQ3zLy6NdiMNwFJ-S3kS-GUA5RzI0p7ibnz-VI9jw=

Use this token to authenticate API requests:
  curl -H 'Authorization: Bearer eyhQ3zLy6NdiMNwFJ-S3kS-GUA5RzI0p7ibnz-VI9jw=' http://localhost:7200/v1/health
```

**Important:** Save the root token securely. You'll need it for all operations.

### 2. Set Encryption Key

```bash
# Generate a secure key
export KEYRAFT_MASTER_KEY=$(openssl rand -base64 32)

# Or use your own (minimum 16 bytes)
export KEYRAFT_MASTER_KEY="your-secure-key-here"
```

### 3. Start Server

```bash
keyrafted start --data-dir ./data --listen :7200
```

The server is now running on `http://localhost:7200`

---

## API Usage

### Authentication

All endpoints (except `/health` and `/metrics`) require Bearer token authentication:

```http
Authorization: Bearer YOUR_TOKEN_HERE
```

### Store Configuration

```bash
curl -X PUT http://localhost:7200/v1/kv/myapp/prod/DB_HOST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"localhost","type":"config"}'
```

### Store Secret (Encrypted)

```bash
curl -X PUT http://localhost:7200/v1/kv/myapp/prod/DB_PASSWORD \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"secret123","type":"secret"}'
```

### Get Value

```bash
curl http://localhost:7200/v1/kv/myapp/prod/DB_HOST \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### List All Keys in Namespace

```bash
curl http://localhost:7200/v1/kv/myapp/prod \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Get Specific Version

```bash
curl http://localhost:7200/v1/kv/myapp/prod/DB_HOST?version=2 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Delete Key

```bash
curl -X DELETE http://localhost:7200/v1/kv/myapp/prod/DB_HOST \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Watch for Changes (Long-Polling)

```bash
curl http://localhost:7200/v1/watch/myapp/prod?timeout=30s \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Watch for Changes (SSE Stream)

```bash
# Using curl with SSE
curl -N http://localhost:7200/v1/watch/myapp/prod?stream=true \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Accept: text/event-stream"
```

**SSE Response Format:**
```
event: connected
data: {"namespace":"myapp/prod","timestamp":"2025-12-20T10:00:00Z"}

event: change
data: {"action":"set","namespace":"myapp/prod","key":"DB_HOST","entry":{...},"timestamp":"2025-12-20T10:01:00Z"}

event: change
data: {"action":"delete","namespace":"myapp/prod","key":"OLD_KEY","timestamp":"2025-12-20T10:02:00Z"}
```

### Create Scoped Token

```bash
curl -X POST http://localhost:7200/v1/auth/token \
  -H "Authorization: Bearer $ROOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scopes": [
      {"namespace": "myapp/*", "read": true, "write": false}
    ],
    "expires_in": 2592000
  }'
```

### Health Check

```bash
curl http://localhost:7200/v1/health
```

**Response:**
```json
{
  "status": "ok",
  "time": "2025-12-20T10:00:00Z",
  "version": "0.1.0"
}
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PUT` | `/v1/kv/{namespace}/{key}` | Set value |
| `GET` | `/v1/kv/{namespace}/{key}` | Get value |
| `GET` | `/v1/kv/{namespace}/{key}?version={n}` | Get specific version |
| `GET` | `/v1/kv/{namespace}` | List keys in namespace |
| `DELETE` | `/v1/kv/{namespace}/{key}` | Delete key |
| `GET` | `/v1/watch/{namespace}?timeout={duration}` | Watch for changes (long-poll) |
| `GET` | `/v1/watch/{namespace}?stream=true` | Watch for changes (SSE stream) |
| `POST` | `/v1/auth/token` | Create token |
| `GET` | `/v1/auth/tokens` | List tokens |
| `DELETE` | `/v1/auth/token/{token}` | Revoke token |
| `GET` | `/v1/namespaces` | List namespaces |
| `GET` | `/v1/health` | Health check |
| `GET` | `/v1/metrics` | Prometheus metrics |

---

## Go Client SDK

### Installation

```bash
go get github.com/keyraft/keyrafted/pkg/client
```

### Basic Usage

```go
import "keyrafted/pkg/client"

// Create client
c := client.NewClient(client.Config{
    BaseURL: "http://localhost:7200",
    Token:   "your-token",
    Timeout: 30 * time.Second,
})

// Store configuration
entry, err := c.Set("myapp/prod", "DB_HOST", "localhost", nil)

// Store secret (encrypted)
_, err = c.SetSecret("myapp/prod", "DB_PASSWORD", "secret123", nil)

// Get value
entry, err := c.Get("myapp/prod", "DB_HOST")
fmt.Println(entry.Value)

// List keys
entries, err := c.List("myapp/prod")
```

### Cached Client (Auto-Reload)

```go
// Create base client
c := client.NewClient(client.Config{
    BaseURL: "http://localhost:7200",
    Token:   "your-token",
})

// Create cached client with auto-reload
cached, err := client.NewCachedClient(client.CacheConfig{
    Client:       c,
    Namespace:    "myapp/prod",
    PollInterval: 10 * time.Second,
})
defer cached.Close()

// Fast reads from cache
value, ok := cached.Get("DB_HOST")

// Register callback for changes
cached.OnChange(func(key, value string) {
    fmt.Printf("Config changed: %s = %s\n", key, value)
    // Reload your application configuration
})
```

### Watch Stream (SSE)

```go
// Stream events using Server-Sent Events (real-time)
events, closeFn, err := c.WatchStream("myapp/prod")
if err != nil {
    log.Fatal(err)
}
defer closeFn()

// Process events as they arrive in real-time
for event := range events {
    fmt.Printf("Change: %s on %s/%s\n", event.Action, event.Namespace, event.Key)
    if event.Entry != nil {
        fmt.Printf("New value: %s\n", event.Entry.Value)
    }
    // Handle config changes immediately
}
```

---

## Docker Usage

### Basic Run

```bash
docker run -d -p 7200:7200 \
  -e KEYRAFT_MASTER_KEY=$(openssl rand -base64 32) \
  -v keyraft-data:/data \
  keyraft/keyrafted:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  keyraft:
    image: keyraft/keyrafted:latest
    ports:
      - "7200:7200"
    environment:
      - KEYRAFT_MASTER_KEY=${KEYRAFT_MASTER_KEY}
    volumes:
      - keyraft-data:/data
    restart: unless-stopped

volumes:
  keyraft-data:
```

### Get Root Token

The container auto-initializes on first run. Get the root token:

```bash
# View full logs (token appears after "Root token" message)
docker logs <container-name>
```

---

## Configuration

### Command Line Flags

```bash
keyrafted start \
  --data-dir /var/lib/keyraft \     # Data directory
  --listen :7200 \                   # Listen address
  --master-key "key-value" \         # Master key (or use env var)
  --master-key-file /path/to/key     # Master key file path
```

### Environment Variables

- `KEYRAFT_MASTER_KEY` - Master encryption key for secrets (32+ bytes recommended)
- `KEYRAFT_DATA_DIR` - Data directory path (overrides `--data-dir` flag)
- `KEYRAFT_LISTEN` - HTTP listen address (overrides `--listen` flag)

---

## Namespace Format

Pattern: `project/environment/service`

**Examples:**
- `billing`
- `billing/prod`
- `billing/prod/api`

**Rules:**
- Alphanumeric, hyphens, underscores
- Maximum 3 levels
- Maximum 256 characters

---

## Security

* **Secrets encrypted at rest** with AES-256-GCM
* **Token-based authentication** with scoped access control
* **Namespace isolation** prevents unauthorized access
* **Version tracking** maintains a full audit trail
* **TLS recommended** for production deployments

**⚠️ Important Security Notes:**
- Always set a strong `KEYRAFT_MASTER_KEY` in production (32+ bytes)
- Store root token securely (password manager, vault, etc.)
- Use scoped tokens for applications (principle of least privilege)
- Enable TLS when exposing to untrusted networks

---

## Community

- **GitHub**: [keyraft/keyrafted](https://github.com/keyraft/keyrafted)
- **Docker Hub**: [keyraft/keyrafted](https://hub.docker.com/r/keyraft/keyrafted)
- **GitHub Packages**: [ghcr.io/keyraft/keyrafted](https://github.com/keyraft/keyrafted/pkgs/container/keyrafted)
- **Issues**: [GitHub Issues](https://github.com/keyraft/keyrafted/issues)
- **Discussions**: [GitHub Discussions](https://github.com/keyraft/keyrafted/discussions)
- **Security**: Report vulnerabilities to xentixar@gmail.com

---

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details

---

**Built with ❤️ for developers who need simple, secure configuration management.**
