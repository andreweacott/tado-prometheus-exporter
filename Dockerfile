# Multi-stage build
FROM golang:1.25 AS builder

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

# Verify binary has correct permissions
RUN chmod 755 /usr/local/bin/tado-exporter

# Switch to non-root user
USER exporter

# Expose metrics port
EXPOSE 9100

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9100/health || exit 1

# Run exporter as non-root user
ENTRYPOINT ["/usr/local/bin/tado-exporter"]
