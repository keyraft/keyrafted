# Getting Started with Keyraft

## Installation

### Build from Source

```bash
git clone https://github.com/keyraft/keyrafted.git
cd keyrafted
go build -o keyrafted
```

## Quick Setup

### 1. Initialize Database

```bash
./keyrafted init --data-dir ./data
```

**Output:**
```
✓ Keyraft initialized successfully!

Root token (save this securely):
  kr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

Use this token to authenticate API requests:
  curl -H 'Authorization: Bearer kr_xxx...' http://localhost:7200/v1/health
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
./keyrafted start --data-dir ./data --listen :7200
```

The server is now running on `http://localhost:7200`

## Basic Usage

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

### Retrieve Value

```bash
curl http://localhost:7200/v1/kv/myapp/prod/DB_HOST \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### List All Keys

```bash
curl http://localhost:7200/v1/kv/myapp/prod \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Configuration

### Command Line Flags

```bash
keyrafted start \
  --data-dir /var/lib/keyraft \     # Data directory
  --listen :7200 \                   # Listen address
  --master-key-file /path/to/key    # Key file path
```

### Environment Variables

- `KEYRAFT_MASTER_KEY` - Master encryption key for secrets

## Next Steps

- [API Reference](api-reference.md) - Complete API documentation
- [Go Client SDK](go-client.md) - Using the Go SDK
- [Deployment Guide](deployment.md) - Production deployment

