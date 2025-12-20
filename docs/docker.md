# Docker Deployment Guide

Complete guide for running Keyraft in Docker.

## Quick Start with Docker Compose

### 1. Create Environment File

```bash
# Generate secure master key
openssl rand -base64 32 > .env
echo "KEYRAFT_MASTER_KEY=$(cat .env)" > .env
```

### 2. Start Keyraft

```bash
docker-compose up -d
```

### 3. Initialize

```bash
docker exec -it keyraft-server /app/keyrafted init --data-dir /data
```

Save the root token!

### 4. Verify

```bash
curl http://localhost:7200/v1/health
```

## Manual Docker Run

### Build Image

```bash
docker build -t keyraft/keyrafted:latest .
```

### Run Container

```bash
# Create volume
docker volume create keyraft-data

# Generate key
export KEYRAFT_MASTER_KEY=$(openssl rand -base64 32)

# Run server
docker run -d \
  --name keyraft \
  -p 7200:7200 \
  -e KEYRAFT_MASTER_KEY="$KEYRAFT_MASTER_KEY" \
  -v keyraft-data:/data \
  --restart unless-stopped \
  keyraft/keyrafted:latest
```

### Initialize

```bash
docker exec keyraft /app/keyrafted init --data-dir /data
```

## Docker Compose Configuration

### Basic Setup

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

### Production Setup with TLS

```yaml
version: '3.8'

services:
  keyraft:
    image: keyraft/keyrafted:latest
    expose:
      - "7200"
    environment:
      - KEYRAFT_MASTER_KEY=${KEYRAFT_MASTER_KEY}
    volumes:
      - keyraft-data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:7200/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - keyraft
    restart: unless-stopped

volumes:
  keyraft-data:
```

## Health Checks

The Docker image includes a built-in health check:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:7200/v1/health || exit 1
```

Check health status:

```bash
docker inspect --format='{{.State.Health.Status}}' keyraft
```

## Data Persistence

### Backup Volume

```bash
# Stop container
docker-compose down

# Backup
docker run --rm \
  -v keyraft-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/keyraft-backup-$(date +%Y%m%d).tar.gz -C /data .
```

### Restore Volume

```bash
# Stop container
docker-compose down

# Restore
docker run --rm \
  -v keyraft-data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/keyraft-backup-YYYYMMDD.tar.gz"

# Start container
docker-compose up -d
```

## Logs

View logs:

```bash
# All logs
docker-compose logs

# Follow logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100
```

## Resource Limits

Add resource constraints:

```yaml
services:
  keyraft:
    # ...
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Multi-Container Stack

Example with application:

```yaml
version: '3.8'

services:
  keyraft:
    image: keyraft/keyrafted:latest
    environment:
      - KEYRAFT_MASTER_KEY=${KEYRAFT_MASTER_KEY}
    volumes:
      - keyraft-data:/data

  app:
    image: myapp:latest
    environment:
      - KEYRAFT_URL=http://keyraft:7200
      - KEYRAFT_TOKEN=${APP_TOKEN}
    depends_on:
      - keyraft

volumes:
  keyraft-data:
```

## Security Best Practices

### 1. Use Secrets Management

```yaml
version: '3.8'

services:
  keyraft:
    image: keyraft/keyrafted:latest
    secrets:
      - keyraft_master_key
    environment:
      - KEYRAFT_MASTER_KEY_FILE=/run/secrets/keyraft_master_key
    volumes:
      - keyraft-data:/data

secrets:
  keyraft_master_key:
    external: true

volumes:
  keyraft-data:
```

Create secret:

```bash
echo "your-master-key" | docker secret create keyraft_master_key -
```

### 2. Run as Non-Root

The image already runs as user `keyraft` (UID 1000).

### 3. Read-Only Root Filesystem

```yaml
services:
  keyraft:
    # ...
    read_only: true
    tmpfs:
      - /tmp
```

### 4. Network Isolation

```yaml
services:
  keyraft:
    networks:
      - backend

networks:
  backend:
    internal: true
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs keyraft

# Check health
docker inspect keyraft

# Run interactive
docker run -it --rm \
  -e KEYRAFT_MASTER_KEY="test" \
  keyraft/keyrafted:latest \
  /app/keyrafted --help
```

### Permission Issues

```bash
# Fix data directory permissions
docker run --rm \
  -v keyraft-data:/data \
  alpine chown -R 1000:1000 /data
```

### Connect from Host

```bash
# Use host network (not recommended for production)
docker run --network=host keyraft/keyrafted:latest
```

## Kubernetes Deployment

Example deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keyraft
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keyraft
  template:
    metadata:
      labels:
        app: keyraft
    spec:
      containers:
      - name: keyraft
        image: keyraft/keyrafted:latest
        ports:
        - containerPort: 7200
        env:
        - name: KEYRAFT_MASTER_KEY
          valueFrom:
            secretKeyRef:
              name: keyraft-secret
              key: master-key
        volumeMounts:
        - name: data
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /v1/health
            port: 7200
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: keyraft-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: keyraft
spec:
  selector:
    app: keyraft
  ports:
  - port: 7200
    targetPort: 7200
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Test with Docker
  run: |
    docker build -t keyraft:test .
    docker run -d --name keyraft-test \
      -e KEYRAFT_MASTER_KEY=test \
      keyraft:test
    
    sleep 5
    docker exec keyraft-test /app/keyrafted init --data-dir /data
    
    # Run tests
    curl http://localhost:7200/v1/health
```

## Performance Tuning

### Optimize Image Size

Multi-stage build already reduces size to ~20MB.

### Persistent Performance

Use local volume driver for better performance:

```yaml
volumes:
  keyraft-data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /path/to/fast/storage
```

