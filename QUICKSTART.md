# Keyraft Quick Start Guide

Get Keyraft up and running in 5 minutes!

## Prerequisites

- Go 1.23 or later
- Linux, macOS, or Windows

## Installation

### Option 1: Build from Source

```bash
git clone https://github.com/keyraft/keyrafted.git
cd keyrafted
go build -o keyrafted
```

### Option 2: Download Binary

```bash
# Coming soon - pre-built binaries
```

## Step 1: Initialize Keyraft

```bash
./keyrafted init --data-dir ./data
```

**Output:**
```
✓ Keyraft initialized successfully!

Root token (save this securely):

  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

Use this token to authenticate API requests:
  curl -H 'Authorization: Bearer eyJhbGc...' http://localhost:7200/v1/health
```

**⚠️ Important:** Save this root token! You'll need it for all API operations.

## Step 2: Set Master Encryption Key

The master key encrypts all secrets stored in Keyraft.

```bash
# Generate a random 32-byte key
export KEYRAFT_MASTER_KEY=$(openssl rand -base64 32)

# Or use your own key (must be at least 16 bytes)
export KEYRAFT_MASTER_KEY="my-super-secure-encryption-key-32bytes"
```

**⚠️ Important:** Store this key securely! Without it, you cannot decrypt your secrets.

## Step 3: Start the Server

```bash
./keyrafted start --data-dir ./data --listen :7200
```

**Output:**
```
Keyraft server starting on :7200
Data directory: ./data
Starting Keyraft server on :7200
```

The server is now running! 🎉

## Step 4: Test the API

Open a new terminal and try these commands:

```bash
# Set your token (replace with your actual token)
export TOKEN="your-root-token-here"

# Health check
curl http://localhost:7200/v1/health

# Store a configuration value
curl -X PUT http://localhost:7200/v1/kv/myapp/prod/DB_HOST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"localhost","type":"config"}'

# Store a secret (encrypted)
curl -X PUT http://localhost:7200/v1/kv/myapp/prod/DB_PASSWORD \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"super-secret","type":"secret"}'

# Retrieve a value
curl http://localhost:7200/v1/kv/myapp/prod/DB_HOST \
  -H "Authorization: Bearer $TOKEN"

# List all keys in namespace
curl http://localhost:7200/v1/kv/myapp/prod \
  -H "Authorization: Bearer $TOKEN"
```

## Step 5: Use the Go SDK

Create a Go program to interact with Keyraft:

```go
package main

import (
    "fmt"
    "keyrafted/pkg/client"
    "log"
    "time"
)

func main() {
    // Create client
    c := client.NewClient(client.Config{
        BaseURL: "http://localhost:7200",
        Token:   "your-root-token-here",
        Timeout: 30 * time.Second,
    })

    // Store configuration
    entry, err := c.Set("myapp/prod", "API_TIMEOUT", "30s", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Stored: %s = %s (v%d)\n", entry.Key, entry.Value, entry.Version)

    // Retrieve configuration
    result, err := c.Get("myapp/prod", "API_TIMEOUT")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Retrieved: %s\n", result.Value)
}
```

## Step 6: Use Cached Client (Recommended)

For production applications, use the cached client for automatic reloading:

```go
package main

import (
    "fmt"
    "keyrafted/pkg/client"
    "log"
    "time"
)

func main() {
    // Create base client
    c := client.NewClient(client.Config{
        BaseURL: "http://localhost:7200",
        Token:   "your-token",
        Timeout: 30 * time.Second,
    })

    // Create cached client
    cached, err := client.NewCachedClient(client.CacheConfig{
        Client:       c,
        Namespace:    "myapp/prod",
        PollInterval: 10 * time.Second, // Check for updates every 10s
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cached.Close()

    // Register callback for config changes
    cached.OnChange(func(key, value string) {
        fmt.Printf("Config changed: %s = %s\n", key, value)
        // Reload your app configuration here
    })

    // Get values from cache (no API calls!)
    if dbHost, ok := cached.Get("DB_HOST"); ok {
        fmt.Printf("DB Host: %s\n", dbHost)
    }

    // Keep running to receive updates
    select {}
}
```

## Common Tasks

### Create a Scoped Token

Only root tokens can create new tokens:

```bash
curl -X POST http://localhost:7200/v1/auth/token \
  -H "Authorization: Bearer $ROOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scopes": [
      {
        "namespace": "myapp/prod",
        "read": true,
        "write": false
      }
    ],
    "metadata": {
      "name": "production-readonly",
      "owner": "team@example.com"
    }
  }'
```

### Watch for Changes

```bash
# Will wait up to 30 seconds for changes
curl http://localhost:7200/v1/watch/myapp/prod?timeout=30s \
  -H "Authorization: Bearer $TOKEN"
```

### Get Version History

```bash
# Get version 2 of a key
curl http://localhost:7200/v1/kv/myapp/prod/DB_HOST?version=2 \
  -H "Authorization: Bearer $TOKEN"
```

### View Metrics

```bash
curl http://localhost:7200/v1/metrics
```

## Production Deployment

### 1. Use Environment Variables

```bash
export KEYRAFT_MASTER_KEY="your-secure-key"
export KEYRAFT_DATA_DIR="/var/lib/keyraft"
export KEYRAFT_LISTEN=":7200"
```

### 2. Run as Systemd Service

Create `/etc/systemd/system/keyrafted.service`:

```ini
[Unit]
Description=Keyraft Server
After=network.target

[Service]
Type=simple
User=keyraft
Group=keyraft
WorkingDirectory=/opt/keyraft
Environment="KEYRAFT_MASTER_KEY=your-key-here"
ExecStart=/opt/keyraft/keyrafted start --data-dir /var/lib/keyraft --listen :7200
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Start the service:

```bash
sudo systemctl enable keyrafted
sudo systemctl start keyrafted
sudo systemctl status keyrafted
```

### 3. Use TLS (Recommended)

Use a reverse proxy like nginx:

```nginx
server {
    listen 443 ssl;
    server_name keyraft.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:7200;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 4. Backup

Regularly backup your data directory:

```bash
# Stop server
sudo systemctl stop keyrafted

# Backup database
tar -czf keyraft-backup-$(date +%Y%m%d).tar.gz /var/lib/keyraft

# Start server
sudo systemctl start keyrafted
```

## Troubleshooting

### Server won't start

**Problem:** `no authentication tokens configured`

**Solution:** Run `keyrafted init` first to create root token.

---

**Problem:** `failed to decrypt secret`

**Solution:** Check that `KEYRAFT_MASTER_KEY` is set correctly. If you lost the key, secrets cannot be recovered.

---

**Problem:** Port already in use

**Solution:** Change the port with `--listen :8080` or kill the process using the port.

### API returns 401 Unauthorized

**Problem:** Authentication failing

**Solution:** 
- Check that you're using the correct token
- Ensure token hasn't expired
- Verify `Authorization: Bearer TOKEN` header format

### Changes not detected by watch

**Problem:** Watch API times out

**Solution:** This is normal. Watch uses long-polling. Changes are only returned when they occur. Reconnect after timeout.

## Next Steps

- Read [API.md](API.md) for complete API documentation
- See [examples/basic_usage.go](examples/basic_usage.go) for more examples
- Check [CONTRIBUTING.md](CONTRIBUTING.md) to contribute
- Join our community (coming soon)

## Getting Help

- 📖 Documentation: [README.md](README.md)
- 🐛 Issues: GitHub Issues
- 💬 Discussions: GitHub Discussions
- 📧 Email: keyrafted@gmail.com

---

**Congratulations!** You're now running Keyraft! 🎉

