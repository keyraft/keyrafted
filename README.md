# Keyraft

[![GitHub release](https://img.shields.io/github/v/release/keyraft/keyrafted?style=flat-square)](https://github.com/keyraft/keyrafted/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/keyraft/keyrafted?style=flat-square)](https://hub.docker.com/r/keyraft/keyrafted)
[![Go Version](https://img.shields.io/github/go-mod/go-version/keyraft/keyrafted?style=flat-square)](https://go.dev/)
[![License](https://img.shields.io/github/license/keyraft/keyrafted?style=flat-square)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/keyraft/keyrafted/ci.yml?branch=main&style=flat-square)](https://github.com/keyraft/keyrafted/actions)

> A lightweight, self-hosted configuration and secrets management system.

Keyraft stores configuration and secrets securely, manages versioning, and provides live updates through a simple HTTP API.

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

## Installation

### Using Docker (Recommended)

```bash
docker pull keyraft/keyrafted:latest
docker run -d -p 7200:7200 \
  -e KEYRAFT_MASTER_KEY=$(openssl rand -base64 32) \
  -v keyraft-data:/data \
  keyraft/keyrafted:latest
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

## Quick Start

```bash
# 1. Initialize
keyrafted init --data-dir ./data

# 2. Start server
export KEYRAFT_MASTER_KEY=$(openssl rand -base64 32)
keyrafted start --data-dir ./data

# 3. Use the API
curl http://localhost:7200/v1/health
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

---

## Documentation

- **[Getting Started](docs/getting-started.md)** - Quick setup guide
- **[API Reference](docs/api-reference.md)** - Complete API documentation  
- **[Go Client SDK](docs/go-client.md)** - Go client library guide
- **[Deployment Guide](docs/deployment.md)** - Production deployment
- **[Docker Guide](docs/docker.md)** - Container deployment
- **[Contributing](docs/CONTRIBUTING.md)** - How to contribute

---

## Community

- **Issues**: [GitHub Issues](https://github.com/keyraft/keyrafted/issues)
- **Discussions**: [GitHub Discussions](https://github.com/keyraft/keyrafted/discussions)
- **Security**: Report vulnerabilities to xentixar@gmail.com

---

## Star History

If you find Keyraft useful, please consider giving it a star ⭐

---

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details

---

## Acknowledgments

Inspired by:
* [etcd](https://etcd.io/) - Distributed key-value store
* [HashiCorp Vault](https://www.vaultproject.io/) - Secrets management
* [Consul](https://www.consul.io/) - Service mesh and configuration

---

**Built with ❤️ for developers who need simple, secure configuration management.**
