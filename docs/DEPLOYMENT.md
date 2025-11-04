# Deployment Guide

Quick deployment guide for tado-prometheus-exporter in homelab environments.

## Table of Contents

1. [Quick Start with Docker Compose](#docker-compose-quick-start)
2. [Standalone Binary](#standalone-deployment)
3. [Docker Container](#docker-container)
4. [Configuration](#configuration)
5. [Troubleshooting](#troubleshooting)

---

## Docker Compose Quick Start

**Recommended for homelab users**: Complete monitoring stack with exporter, Prometheus, and Grafana in one command.

```bash
# Clone repository
git clone https://github.com/andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter

# Create .env file
cat > .env <<EOF
TADO_TOKEN_PASSPHRASE=your-secure-passphrase-here
COMPOSE_PROJECT_NAME=tado-monitoring
EOF

# Start services
docker-compose up -d

# Check status
docker-compose ps
```

### Access Services

- **Exporter metrics**: http://localhost:9100/metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (login: admin/admin)

### View Metrics

```bash
# Direct metrics endpoint
curl http://localhost:9100/metrics | grep tado_

# Query in Prometheus web UI
# - Query: tado_temperature_measured_celsius
# - Label filters: zone_name="Bedroom"

# Grafana dashboard
# - Login: admin / admin
# - Add Prometheus data source (http://prometheus:9090)
# - Import dashboard or create custom queries
```

### Docker Compose Commands

```bash
# View logs
docker-compose logs -f exporter

# Stop services
docker-compose down

# View logs for specific service
docker-compose logs -f prometheus

# Update services
docker-compose pull && docker-compose up -d
```

---

## Standalone Deployment

### Prerequisites

- Go 1.25.1 or later
- Linux, macOS, or Windows
- Network connectivity to Tado API

### Installation

#### Option 1: Build from Source

```bash
git clone https://github.com/andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter
go build -o tado-exporter ./cmd/exporter
```

#### Option 2: Download Pre-built Binary

```bash
# Download latest release from GitHub Releases
wget https://github.com/andreweacott/tado-prometheus-exporter/releases/download/v1.0.0/tado-prometheus-exporter
chmod +x tado-prometheus-exporter
```

### Initial Setup

1. **Create token storage directory:**
   ```bash
   mkdir -p ~/.tado-exporter
   chmod 700 ~/.tado-exporter
   ```

2. **Set passphrase environment variable:**
   ```bash
   export TADO_TOKEN_PASSPHRASE="your-secure-passphrase-here"
   ```

3. **Run the exporter:**
   ```bash
   ./tado-exporter --token-path ~/.tado-exporter/token.json \
                   --token-passphrase "$TADO_TOKEN_PASSPHRASE" \
                   --port 9100
   ```

4. **First run - Device Code Flow:**
   - Exporter will display device code and verification URL
   - Visit the URL in your browser
   - Follow Tado authorization prompts
   - Token will be encrypted and saved automatically

### Usage

```bash
./tado-exporter [FLAGS]

Flags:
  --port               HTTP server port (default: 9100)
  --token-path         Path to encrypted token file (default: ~/.tado-exporter/token.json)
  --token-passphrase   Passphrase for token encryption (required)
  --home-id            Optional home ID filter
  --scrape-timeout     API request timeout in seconds (default: 10)
  --log-level          Logging verbosity: debug, info, warn, error (default: info)
```

### Example: Standalone with Prometheus

1. **Create `prometheus.yml`:**
   ```yaml
   global:
     scrape_interval: 60s

   scrape_configs:
     - job_name: 'tado-exporter'
       static_configs:
         - targets: ['localhost:9100']
   ```

2. **Start Prometheus:**
   ```bash
   prometheus --config.file=prometheus.yml
   ```

3. **Access Prometheus:**
   - http://localhost:9090
   - Query: `tado_temperature_measured_celsius`

---

## Docker Container

### Prerequisites

- Docker Engine 20.10+

### Quick Start

```bash
# Build image locally
docker build -t tado-prometheus-exporter:latest .

# Create token storage volume
docker volume create tado-tokens

# Run container
docker run -d \
  --name tado-exporter \
  -p 9100:9100 \
  -v tado-tokens:/root/.tado-exporter \
  -e TADO_TOKEN_PASSPHRASE="your-secure-passphrase" \
  tado-prometheus-exporter:latest \
  --token-path /root/.tado-exporter/token.json \
  --token-passphrase "$TADO_TOKEN_PASSPHRASE" \
  --port 9100
```

### Health Check

```bash
# Test health endpoint
curl http://localhost:9100/health

# Test metrics endpoint
curl http://localhost:9100/metrics
```

### Container Logs

```bash
# View logs
docker logs tado-exporter

# Follow logs
docker logs -f tado-exporter

# View last 50 lines
docker logs --tail 50 tado-exporter
```

### Docker Hub

```bash
# Pull from Docker Hub (when published)
docker pull andreweacott/tado-prometheus-exporter:latest

# Run from Docker Hub
docker run -d \
  --name tado-exporter \
  -p 9100:9100 \
  -v tado-tokens:/root/.tado-exporter \
  -e TADO_TOKEN_PASSPHRASE="your-passphrase" \
  andreweacott/tado-prometheus-exporter:latest
```

---

## Configuration

### Environment Variables

All configuration can be set via environment variables (useful for containers):

| Variable | Default | Description |
|----------|---------|-------------|
| `TADO_TOKEN_PATH` | `~/.tado-exporter/token.json` | Token file location |
| `TADO_TOKEN_PASSPHRASE` | - | Encryption passphrase (required) |
| `TADO_PORT` | `9100` | HTTP server port |
| `TADO_HOME_ID` | - | Optional home ID filter |
| `TADO_SCRAPE_TIMEOUT` | `10` | API timeout in seconds |
| `TADO_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

### Command-Line Flags

```bash
--port int                  HTTP server port (default 9100)
--token-path string         Path to token file (default ~/.tado-exporter/token.json)
--token-passphrase string   Passphrase for token encryption (required)
--home-id string            Optional home ID filter
--scrape-timeout int        API request timeout in seconds (default 10)
--log-level string          Log level: debug, info, warn, error (default info)
```

### Configuration Priority

1. Command-line flags (highest priority)
2. Environment variables
3. Default values (lowest priority)

---

## Maintenance

### Token Rotation

Token is automatically refreshed by the clambin/tado library. If needed:

```bash
# Remove encrypted token file
rm ~/.tado-exporter/token.json

# Restart exporter (will re-authenticate)
systemctl restart tado-exporter  # if using systemd
docker restart tado-exporter      # if using Docker
```

### Monitoring the Exporter

```bash
# Check if exporter is running
curl -s http://localhost:9100/health | jq .

# Sample output:
# {
#   "status": "healthy",
#   "uptime": "2h30m15s",
#   "timestamp": "2025-11-04T10:15:30Z"
# }
```

---

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and solutions.

### Quick Diagnostics

```bash
# Check if exporter is listening
netstat -tlnp | grep 9100

# Test metrics endpoint
curl -v http://localhost:9100/metrics | head -30

# Check token file exists
ls -la ~/.tado-exporter/

# Increase log level for debugging
# Set TADO_LOG_LEVEL=debug and restart

# Check Tado API connectivity
curl -v https://api.tado.com
```

---

## Support

For deployment issues:

1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
2. Review logs: `docker logs` or `journalctl`
3. Open issue on GitHub: https://github.com/andreweacott/tado-prometheus-exporter/issues
4. Check ARCHITECTURE.md for design details

---

## Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues
- [HTTP_ENDPOINTS.md](HTTP_ENDPOINTS.md) - API reference
- [README.md](../README.md) - Quick start
