# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o keyrafted .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and entrypoint from builder
COPY --from=builder /build/keyrafted /app/keyrafted
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

# Create data directory
RUN mkdir -p /data && chmod 700 /data

# Make entrypoint executable
RUN chmod +x /app/docker-entrypoint.sh

# Expose port
EXPOSE 7200

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:7200/v1/health || exit 1

# Run as non-root user
RUN adduser -D -u 1000 keyraft && chown -R keyraft:keyraft /app /data
USER keyraft

# Use entrypoint script
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["--data-dir", "/data", "--listen", ":7200"]

