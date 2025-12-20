# Changelog

All notable changes to Keyraft will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-20

### Added

#### Core Features
- **Key-value store** with BoltDB backend
- **AES-256-GCM encryption** for secrets at rest
- **Namespaces** for multi-tenancy (pattern: `project/environment/service`)
- **Token-based authentication** with scoped access control
- **Version tracking** - Complete history of all changes
- **Watch API** - Long-polling for live configuration updates
- **HTTP/JSON API** - RESTful interface with 11 endpoints

#### Server
- CLI tool with `init` and `start` commands
- Configurable data directory and listen address
- Master encryption key from environment variable or file
- Graceful shutdown handling
- Request logging middleware
- Prometheus metrics endpoint (`/v1/metrics`)
- Health check endpoint (`/v1/health`)

#### API Endpoints
- `PUT /v1/kv/{namespace}/{key}` - Set key value
- `GET /v1/kv/{namespace}/{key}` - Get key value
- `GET /v1/kv/{namespace}/{key}?version=N` - Get specific version
- `DELETE /v1/kv/{namespace}/{key}` - Delete key (soft delete)
- `GET /v1/kv/{namespace}` - List all keys in namespace
- `GET /v1/watch/{namespace}` - Watch for changes
- `GET /v1/namespaces` - List all namespaces
- `POST /v1/auth/token` - Create authentication token
- `GET /v1/auth/tokens` - List all tokens
- `DELETE /v1/auth/token/{token}` - Revoke token

#### Go Client SDK
- `client.NewClient()` - Basic HTTP client
- `client.NewCachedClient()` - Cached client with auto-reload
- Methods: `Set()`, `SetSecret()`, `Get()`, `GetVersion()`, `Delete()`, `List()`, `Watch()`
- Change callbacks via `OnChange()`
- Thread-safe operations
- Connection pooling and retry logic

#### Docker Support
- Multi-stage Dockerfile (final size: ~20MB)
- Docker Compose configuration
- Health checks built-in
- Non-root user (UID 1000)
- Volume persistence

#### Development Tools
- Makefile with common tasks
- GitHub Actions CI/CD workflow
- Unit tests (models, crypto)
- Integration tests (storage, auth, engine, watch)
- Example web application with live config reload

#### Documentation
- Getting Started guide
- Complete API reference
- Go Client SDK guide
- Deployment guide (systemd, nginx, monitoring)
- Docker deployment guide

### Security
- Secrets encrypted at rest with AES-256-GCM
- Token-based authentication with Bearer tokens
- Scoped access control (read/write per namespace)
- Namespace isolation
- Version tracking for audit trail
- Root token with full privileges
- PBKDF2 key derivation

### Project Statistics
- **20 Go files** (~4,500 lines of code)
- **7 documentation files**
- **Binary size:** 9.4MB
- **Test coverage:** Unit + Integration tests
- **Dependencies:** 4 (BoltDB, Gorilla Mux, Cobra, Go Crypto)

### Known Limitations
- Single-node only (clustering planned for v0.3)
- Long-polling for watch (SSE planned for v0.2)
- No rate limiting (planned for v0.2)
- No web UI (API-first design)
- No TLS built-in (use reverse proxy)

### Notes
- This is the initial MVP release
- Production-ready for single-node deployments
- All core features functional and tested

## [Unreleased]

### Planned for v0.2
- SSE/WebSocket streaming for watch
- Enhanced audit logging with query API
- Rate limiting (per-IP, per-token)
- Docker image on Docker Hub
- PHP and Node.js SDKs
- Metrics dashboard

### Planned for v0.3
- Raft-based clustering
- Multi-node high availability
- Leader election
- Data replication

### Planned for v1.0
- Production-ready with guarantees
- Backward compatibility promise
- Performance optimizations
- Comprehensive benchmarks
- Complete audit logging

---

[0.1.0]: https://github.com/keyraft/keyrafted/releases/tag/v0.1.0

