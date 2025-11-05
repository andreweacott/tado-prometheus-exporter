# Multi-stage build
FROM golang:1.24 AS builder

WORKDIR /app

# Copy module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o tado-exporter ./cmd/exporter

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates wget

# Create non-root user for security
# UID 1000 is a safe non-root UID that doesn't conflict with system users
RUN addgroup -g 1000 -S exporter && \
    adduser -u 1000 -S -G exporter -h /home/exporter -s /sbin/nologin exporter && \
    mkdir -p /home/exporter/.config /home/exporter/.tado-exporter && \
    chown -R exporter:exporter /home/exporter

# Copy binary from builder
COPY --from=builder --chown=exporter:exporter /app/tado-exporter /usr/local/bin/tado-exporter

# Copy entrypoint script - runs as root to fix volume permissions, then drops to exporter user
COPY <<'EOF' /usr/local/bin/entrypoint.sh
#!/bin/sh
set -e

# Ensure token directory exists with proper permissions
# This handles cases where volume mounts override the Dockerfile's setup
mkdir -p /home/exporter/.tado-exporter || { echo "ERROR: Failed to create token directory"; exit 1; }
chown exporter:exporter /home/exporter/.tado-exporter || { echo "ERROR: Failed to fix directory ownership"; exit 1; }
chmod 755 /home/exporter/.tado-exporter || { echo "ERROR: Failed to set directory permissions"; exit 1; }

# Secure token file permissions if it already exists from a previous run
if [ -f /home/exporter/.tado-exporter/token.json ]; then
    chmod 600 /home/exporter/.tado-exporter/token.json || { echo "WARNING: Could not secure token file"; }
fi

# Drop to exporter user and run the application
exec su-exec exporter /usr/local/bin/tado-exporter "$@"
EOF

# Install su-exec for dropping privileges safely
RUN apk --no-cache add su-exec

# Verify binary has correct permissions
RUN chmod 755 /usr/local/bin/tado-exporter /usr/local/bin/entrypoint.sh

# Expose metrics port
EXPOSE 9100

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9100/health || exit 1

# Run entrypoint as root (needed to fix volume mount permissions)
# The entrypoint will drop privileges to exporter user
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
