# Deployment Guide

Production deployment guide for Keyraft.

## System Requirements

- Linux, macOS, or Windows
- Go 1.23+ (for building from source)
- 100MB disk space
- 256MB RAM minimum

## Production Setup

### 1. Build Binary

```bash
git clone https://github.com/keyraft/keyrafted.git
cd keyrafted
CGO_ENABLED=0 go build -ldflags="-s -w" -o keyrafted
```

### 2. Create System User

```bash
sudo useradd -r -s /bin/false keyraft
sudo mkdir -p /var/lib/keyraft
sudo chown keyraft:keyraft /var/lib/keyraft
```

### 3. Install Binary

```bash
sudo cp keyrafted /usr/local/bin/
sudo chmod +x /usr/local/bin/keyrafted
```

### 4. Generate Master Key

```bash
# Generate strong encryption key
openssl rand -base64 32 > /etc/keyraft/master.key
sudo chmod 600 /etc/keyraft/master.key
sudo chown keyraft:keyraft /etc/keyraft/master.key
```

### 5. Initialize Database

```bash
sudo -u keyraft keyrafted init --data-dir /var/lib/keyraft
```

Save the root token securely (password manager, vault, etc.).

---

## Systemd Service

### Create Service File

Create `/etc/systemd/system/keyrafted.service`:

```ini
[Unit]
Description=Keyraft Configuration Server
After=network.target
Documentation=https://github.com/keyraft/keyrafted

[Service]
Type=simple
User=keyraft
Group=keyraft
WorkingDirectory=/var/lib/keyraft

# Load encryption key
Environment="KEYRAFT_MASTER_KEY_FILE=/etc/keyraft/master.key"

# Start server
ExecStart=/usr/local/bin/keyrafted start \
    --data-dir /var/lib/keyraft \
    --listen :7200 \
    --master-key-file /etc/keyraft/master.key

# Restart policy
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/keyraft

# Limits
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable keyrafted
sudo systemctl start keyrafted
sudo systemctl status keyrafted
```

### View Logs

```bash
sudo journalctl -u keyrafted -f
```

---

## Reverse Proxy (Nginx)

### Setup TLS with Nginx

Create `/etc/nginx/sites-available/keyraft`:

```nginx
upstream keyraft {
    server 127.0.0.1:7200;
    keepalive 32;
}

server {
    listen 80;
    server_name your-domain.com;  # Replace with your domain
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;  # Replace with your domain

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;

    # Logging
    access_log /var/log/nginx/keyraft-access.log;
    error_log /var/log/nginx/keyraft-error.log;

    # Proxy to Keyraft
    location / {
        proxy_pass http://keyraft;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Long-polling support
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
        
        # Connection reuse
        proxy_set_header Connection "";
    }

    # Health check endpoint
    location /v1/health {
        proxy_pass http://keyraft;
        access_log off;
    }
}
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/keyraft /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## Backup Strategy

### Backup Script

Create `/usr/local/bin/backup-keyraft.sh`:

```bash
#!/bin/bash

BACKUP_DIR="/backup/keyraft"
DATA_DIR="/var/lib/keyraft"
DATE=$(date +%Y%m%d_%H%M%S)

# Stop server
systemctl stop keyrafted

# Create backup
mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/keyraft-$DATE.tar.gz" \
    -C "$DATA_DIR" .

# Start server
systemctl start keyrafted

# Keep last 30 days
find "$BACKUP_DIR" -name "keyraft-*.tar.gz" -mtime +30 -delete

echo "Backup completed: keyraft-$DATE.tar.gz"
```

Make executable:

```bash
sudo chmod +x /usr/local/bin/backup-keyraft.sh
```

### Automated Backups

Add to crontab:

```bash
# Daily backup at 2 AM
0 2 * * * /usr/local/bin/backup-keyraft.sh >> /var/log/keyraft-backup.log 2>&1
```

### Restore from Backup

```bash
# Stop server
sudo systemctl stop keyrafted

# Restore data
sudo tar -xzf /backup/keyraft/keyraft-20250120_020000.tar.gz \
    -C /var/lib/keyraft

# Fix permissions
sudo chown -R keyraft:keyraft /var/lib/keyraft

# Start server
sudo systemctl start keyrafted
```

---

## Monitoring

### Health Check

```bash
curl -f http://localhost:7200/v1/health || exit 1
```

### Prometheus Metrics

```bash
curl http://localhost:7200/v1/metrics
```

Add to Prometheus config:

```yaml
scrape_configs:
  - job_name: 'keyraft'
    static_configs:
      - targets: ['localhost:7200']
    metrics_path: '/v1/metrics'
```

### Alerting

Example alert rules:

```yaml
groups:
  - name: keyraft
    rules:
      - alert: KeyraftDown
        expr: up{job="keyraft"} == 0
        for: 5m
        annotations:
          summary: "Keyraft server is down"

      - alert: KeyraftHighWatches
        expr: keyraft_active_watches > 1000
        for: 10m
        annotations:
          summary: "Too many active watches"
```

---

## Security Hardening

### 1. Firewall Rules

```bash
# Allow only necessary access
sudo ufw allow 443/tcp  # HTTPS only
sudo ufw enable
```

### 2. Token Management

```bash
# Create scoped tokens for applications
curl -X POST http://localhost:7200/v1/auth/token \
  -H "Authorization: Bearer $ROOT_TOKEN" \
  -d '{
    "scopes": [
      {"namespace": "myapp/*", "read": true, "write": false}
    ],
    "expires_in": 2592000
  }'
```

### 3. Regular Key Rotation

```bash
# Generate new master key
NEW_KEY=$(openssl rand -base64 32)

# Update key file
echo "$NEW_KEY" | sudo tee /etc/keyraft/master.key

# Restart service
sudo systemctl restart keyrafted
```

**Note:** Existing secrets will need to be re-encrypted with the new key.

### 4. Audit Logs

Monitor access logs:

```bash
sudo journalctl -u keyrafted | grep "GET\|PUT\|DELETE"
```

---

## Performance Tuning

### System Limits

```bash
# Increase file descriptors
sudo sysctl -w fs.file-max=100000

# Persist setting
echo "fs.file-max = 100000" | sudo tee -a /etc/sysctl.conf
```

### BoltDB Optimization

For high-write workloads, consider SSD storage for better performance.

---

## Troubleshooting

### Server Won't Start

```bash
# Check logs
sudo journalctl -u keyrafted -n 50

# Verify permissions
ls -la /var/lib/keyraft

# Test manually
sudo -u keyraft /usr/local/bin/keyrafted start \
    --data-dir /var/lib/keyraft \
    --listen :7200
```

### High Memory Usage

```bash
# Check process
ps aux | grep keyrafted

# Monitor in real-time
top -p $(pgrep keyrafted)
```

### Database Corruption

```bash
# Stop service
sudo systemctl stop keyrafted

# Restore from backup
sudo tar -xzf /backup/keyraft/latest.tar.gz -C /var/lib/keyraft

# Start service
sudo systemctl start keyrafted
```

---

## Scaling Considerations

Current version (v0.1) is single-node only.

For high-availability:
- Wait for v0.3 with Raft clustering
- Use database replication backups
- Deploy to multiple regions with DNS failover
- Monitor and alert on downtime

---

## Update Process

```bash
# Download new binary
wget https://github.com/keyraft/keyrafted/releases/download/v0.2.0/keyrafted

# Stop service
sudo systemctl stop keyrafted

# Backup current
sudo cp /usr/local/bin/keyrafted /usr/local/bin/keyrafted.old

# Install new version
sudo cp keyrafted /usr/local/bin/
sudo chmod +x /usr/local/bin/keyrafted

# Start service
sudo systemctl start keyrafted

# Check status
sudo systemctl status keyrafted
```

