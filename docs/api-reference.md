# API Reference

Base URL: `http://localhost:7200/v1`

## Authentication

All endpoints (except `/health` and `/metrics`) require Bearer token authentication:

```http
Authorization: Bearer YOUR_TOKEN_HERE
```

## Key-Value Operations

### Set Value

Store or update a configuration value or secret.

```http
PUT /v1/kv/{namespace}/{key}
```

**Request Body:**
```json
{
  "value": "localhost",
  "type": "config",
  "metadata": {
    "env": "production"
  }
}
```

**Parameters:**
- `type` - Either `config` (plaintext) or `secret` (encrypted)
- `metadata` - Optional key-value metadata

**Example:**
```bash
curl -X PUT http://localhost:7200/v1/kv/billing/prod/DB_HOST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"db.example.com","type":"config"}'
```

**Response:**
```json
{
  "namespace": "billing/prod",
  "key": "DB_HOST",
  "value": "db.example.com",
  "type": "config",
  "version": 1,
  "created_at": "2025-12-20T10:00:00Z",
  "updated_at": "2025-12-20T10:00:00Z"
}
```

---

### Get Value

Retrieve the latest version of a key.

```http
GET /v1/kv/{namespace}/{key}
```

**Example:**
```bash
curl http://localhost:7200/v1/kv/billing/prod/DB_HOST \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Specific Version

```http
GET /v1/kv/{namespace}/{key}?version={version_number}
```

**Example:**
```bash
curl http://localhost:7200/v1/kv/billing/prod/DB_HOST?version=2 \
  -H "Authorization: Bearer $TOKEN"
```

---

### List Keys

List all keys in a namespace.

```http
GET /v1/kv/{namespace}
```

**Example:**
```bash
curl http://localhost:7200/v1/kv/billing/prod \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "namespace": "billing/prod",
  "count": 3,
  "keys": [
    {
      "key": "DB_HOST",
      "value": "db.example.com",
      "type": "config",
      "version": 1
    }
  ]
}
```

---

### Delete Key

Soft delete a key (versions are preserved).

```http
DELETE /v1/kv/{namespace}/{key}
```

---

### Watch for Changes

Long-polling endpoint to watch for changes in a namespace.

```http
GET /v1/watch/{namespace}?timeout=30s
```

**Parameters:**
- `timeout` - How long to wait for changes (default: 30s)

**Response on change:**
```json
{
  "action": "set",
  "namespace": "billing/prod",
  "key": "DB_HOST",
  "timestamp": "2025-12-20T10:30:00Z"
}
```

**Response on timeout:**
```json
{
  "timeout": true
}
```

---

## Authentication Operations

### Create Token

Create a new authentication token (requires root token).

```http
POST /v1/auth/token
```

**Request Body:**
```json
{
  "scopes": [
    {
      "namespace": "billing/prod",
      "read": true,
      "write": false
    }
  ],
  "expires_in": 86400,
  "metadata": {
    "name": "production-readonly"
  }
}
```

**Scope Patterns:**
- `billing/prod` - Exact match
- `billing/*` - All namespaces starting with `billing/`
- `*` - All namespaces (wildcard)

**Example:**
```bash
curl -X POST http://localhost:7200/v1/auth/token \
  -H "Authorization: Bearer $ROOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scopes": [
      {"namespace": "myapp/*", "read": true, "write": false}
    ]
  }'
```

---

### List Tokens

```http
GET /v1/auth/tokens
```

Requires root token.

---

### Revoke Token

```http
DELETE /v1/auth/token/{token}
```

Requires root token.

---

## Namespace Operations

### List Namespaces

```http
GET /v1/namespaces
```

### Get Namespace

```http
GET /v1/namespaces/{namespace}
```

---

## System Operations

### Health Check

```http
GET /v1/health
```

No authentication required.

**Response:**
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

---

### Metrics

```http
GET /v1/metrics
```

Returns Prometheus-compatible metrics. No authentication required.

---

## Data Model

### Namespace Format

Pattern: `project/environment/service`

**Examples:**
- `billing`
- `billing/prod`
- `billing/prod/api`

**Rules:**
- Alphanumeric, hyphens, underscores
- Maximum 3 levels
- Maximum 256 characters

### Key Format

**Examples:**
- `DB_HOST`
- `api.timeout`
- `feature-flag-v2`

**Rules:**
- Alphanumeric, dots, hyphens, underscores
- Maximum 256 characters

### Entry Types

- `config` - Plaintext configuration
- `secret` - Encrypted secret (AES-256-GCM)

---

## Error Responses

**401 Unauthorized:**
```json
{
  "error": "invalid or expired token"
}
```

**403 Forbidden:**
```json
{
  "error": "insufficient permissions"
}
```

**404 Not Found:**
```json
{
  "error": "key not found"
}
```

**400 Bad Request:**
```json
{
  "error": "invalid namespace format"
}
```

---

## Complete Example

```bash
# Set token
export TOKEN="your-token-here"

# Store config
curl -X PUT http://localhost:7200/v1/kv/myapp/prod/API_URL \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"https://api.example.com","type":"config"}'

# Store secret
curl -X PUT http://localhost:7200/v1/kv/myapp/prod/API_KEY \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"sk_live_xxxxx","type":"secret"}'

# Get config
curl http://localhost:7200/v1/kv/myapp/prod/API_URL \
  -H "Authorization: Bearer $TOKEN"

# List all
curl http://localhost:7200/v1/kv/myapp/prod \
  -H "Authorization: Bearer $TOKEN"

# Watch for changes
curl http://localhost:7200/v1/watch/myapp/prod?timeout=60s \
  -H "Authorization: Bearer $TOKEN"
```

