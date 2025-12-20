# Keyrafted

> **Keyrafted** is the server component of the Keyraft project — a lightweight, self-hosted configuration and secrets management system.

Keyrafted stores configuration and secrets securely, manages versioning, and serves requests from clients, CLI, and SDKs.

---

## Features (v0.1)

* ✅ Key-value store with versioning
* ✅ Encrypted storage (AES-256-GCM) for secrets
* ✅ Namespaces for isolation (`project/environment/service`)
* ✅ Token-based authentication with scoped API keys
* ✅ Historical version tracking
* ✅ Watch API for live updates (long-polling)
* ✅ HTTP/JSON protocol
* ✅ Prometheus metrics endpoint
* ✅ Go SDK with caching and auto-reload

Planned Features (v0.2+):

* SSE/WebSocket streaming for watch updates
* Role-based access control (RBAC)
* Audit logging
* Raft-based distributed clustering
* Multi-region high availability
* gRPC API support

---

## Architecture

Single-node architecture for v0.1 (clustering planned for v0.3):

```
       +------------------------+
       |   Client / CLI / SDK   |
       +-----------+------------+
                   |
                   v
          +------------------+
          |  Keyrafted Server |
          |  HTTP API Layer   |
          +--------+---------+
                   |
                   v
          +------------------+
          |  BoltDB Storage  |
          +------------------+
```

---

## Quick Start

```bash
# Build
git clone https://github.com/keyraft/keyrafted.git
cd keyrafted
go build -o keyrafted

# Initialize (generates root token)
./keyrafted init --data-dir ./data

# Start server
export KEYRAFT_MASTER_KEY=$(openssl rand -base64 32)
./keyrafted start --data-dir ./data --listen :7200
```

**📖 See [Getting Started Guide](docs/getting-started.md) for detailed setup**

---

## API Endpoints

- `PUT /v1/kv/{namespace}/{key}` - Set value
- `GET /v1/kv/{namespace}/{key}` - Get value
- `GET /v1/kv/{namespace}` - List keys
- `DELETE /v1/kv/{namespace}/{key}` - Delete key
- `GET /v1/watch/{namespace}` - Watch changes
- `POST /v1/auth/token` - Create token
- `GET /v1/health` - Health check
- `GET /v1/metrics` - Metrics

See [API Reference](docs/api-reference.md) for complete documentation.

---

## Go Client SDK

```go
import "keyrafted/pkg/client"

// Create client
c := client.NewClient(client.Config{
    BaseURL: "http://localhost:7200",
    Token:   "your-token",
})

// Set and get
c.Set("myapp/prod", "DB_HOST", "localhost", nil)
entry, _ := c.Get("myapp/prod", "DB_HOST")

// Cached client with auto-reload
cached, _ := client.NewCachedClient(client.CacheConfig{
    Client:       c,
    Namespace:    "myapp/prod",
    PollInterval: 10 * time.Second,
})
defer cached.Close()

// Register callback for changes
cached.OnChange(func(key, value string) {
    // Handle config changes
})
```

**📖 See [Go Client Documentation](docs/go-client.md) for a complete guide**

---

## Security

* **Secrets encrypted at rest** with AES-256-GCM
* **Token-based authentication** with scoped access control
* **Namespace isolation** prevents unauthorized access
* **Version tracking** maintains a full audit trail
* **TLS recommended** for production deployments

**⚠️ Important Security Notes:**
- Always set a strong `KEYRAFT_MASTER_KEY` in production
- Store root token securely (password manager, vault, etc.)
- Use scoped tokens for applications (principle of least privilege)
- Enable TLS when exposing to untrusted networks

---

## Development

### Running Tests

```bash
# Unit tests
go test ./tests/unit -v

# Integration tests
go test ./tests/integration -v

# All tests with coverage
go test ./tests/... -cover
```

### Project Structure

```
keyrafted/
├── cmd/                  # CLI commands (root, init, start)
├── internal/
│   ├── api/             # HTTP API handlers
│   ├── auth/            # Authentication service
│   ├── crypto/          # Encryption utilities
│   ├── engine/          # Config/secrets engine
│   ├── models/          # Data models
│   ├── storage/         # BoltDB storage layer
│   └── watch/           # Watch manager
├── pkg/
│   └── client/          # Go SDK
├── tests/
│   ├── unit/            # Unit tests
│   └── integration/     # Integration tests
└── docs/                # Documentation
```

---

## Roadmap

**v0.1** ✅ (Current)
- Single-node server
- BoltDB storage
- Token authentication
- HTTP API
- Versioning & watch
- Go SDK

**v0.2** (Next)
- SSE for watch streaming
- Enhanced audit logging
- Metrics dashboard
- Docker image

**v0.3** (Future)
- Raft clustering
- Multi-node HA
- Leader election

**v1.0** (Stable)
- Production-ready
- Backward compatibility guarantee
- Performance optimizations

---

## Documentation

- **[Getting Started](docs/getting-started.md)** - Quick setup guide
- **[API Reference](docs/api-reference.md)** - Complete API documentation  
- **[Go Client SDK](docs/go-client.md)** - Go client library guide
- **[Deployment Guide](docs/deployment.md)** - Production deployment

---

## License

Apache License 2.0 – See [LICENSE](LICENSE) for details

---

## Acknowledgments

Inspired by:
* etcd - Distributed key-value store
* HashiCorp Vault - Secrets management
* Consul - Service mesh and configuration
* Apache ZooKeeper – Distributed coordination
