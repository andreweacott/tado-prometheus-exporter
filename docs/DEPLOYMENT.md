# Deployment Guide

Complete deployment guide for tado-prometheus-exporter, covering standalone, Docker, Docker Compose, and Kubernetes deployments.

## Table of Contents

1. [Standalone Deployment](#standalone-deployment)
2. [Docker Container](#docker-container)
3. [Docker Compose](#docker-compose)
4. [Kubernetes](#kubernetes)
5. [Systemd Service](#systemd-service)
6. [Configuration](#configuration)
7. [Troubleshooting](#troubleshooting)

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
- Docker Compose (optional, for full stack)

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
# Check container health
docker inspect tado-exporter --format='{{.State.Health.Status}}'

# Test metrics endpoint
curl http://localhost:9100/metrics

# Test health endpoint
curl http://localhost:9100/health
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

### Image Variants

- `tado-prometheus-exporter:latest` - Latest release
- `tado-prometheus-exporter:v1.0.0` - Specific version
- `tado-prometheus-exporter:dev` - Development build (if available)

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

## Docker Compose

### Quick Start

Complete monitoring stack with exporter, Prometheus, and Grafana:

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

- **Exporter**: http://localhost:9100/metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

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

### Docker Compose Services

#### Exporter
- Container: `tado-exporter`
- Port: `9100`
- Volumes: `tado-tokens` (encrypted token storage)
- Health: Checked every 30 seconds

#### Prometheus
- Container: `prometheus`
- Port: `9090`
- Volumes: `prometheus-data` (persistent storage)
- Config: `prometheus.yml` (local file)
- Scrape interval: 60 seconds

#### Grafana
- Container: `grafana`
- Port: `3000`
- Volumes: `grafana-data` (persistent storage)
- Default credentials: admin / admin

### Docker Compose Commands

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# View logs for all services
docker-compose logs -f

# View logs for specific service
docker-compose logs -f exporter

# Rebuild images
docker-compose build

# Remove volumes (WARNING: deletes data)
docker-compose down -v

# Update services (pull new images)
docker-compose pull && docker-compose up -d
```

### Configuration

Edit `docker-compose.yml` to customize:

```yaml
services:
  exporter:
    # Change port
    ports:
      - "9101:9100"  # Host:container

    # Change scrape timeout
    environment:
      TADO_SCRAPE_TIMEOUT: "15"

    # Add optional home ID filter
    command:
      - --home-id=12345
```

---

## Kubernetes

### Prerequisites

- Kubernetes 1.20+
- kubectl configured
- Persistent storage available (optional)

### Deployment Manifest

Create `tado-exporter-k8s.yaml`:

```yaml
---
# Namespace
apiVersion: v1
kind: Namespace
metadata:
  name: tado-monitoring

---
# ConfigMap for Prometheus configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: tado-monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 60s
    scrape_configs:
      - job_name: 'tado-exporter'
        static_configs:
          - targets: ['tado-exporter:9100']

---
# Secret for token passphrase
apiVersion: v1
kind: Secret
metadata:
  name: tado-passphrase
  namespace: tado-monitoring
type: Opaque
stringData:
  passphrase: "your-secure-passphrase-here"

---
# PersistentVolumeClaim for token storage
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: tado-tokens
  namespace: tado-monitoring
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi

---
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tado-exporter
  namespace: tado-monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tado-exporter
  template:
    metadata:
      labels:
        app: tado-exporter
    spec:
      containers:
      - name: tado-exporter
        image: andreweacott/tado-prometheus-exporter:latest
        imagePullPolicy: IfNotPresent
        ports:
        - name: metrics
          containerPort: 9100
        env:
        - name: TADO_TOKEN_PASSPHRASE
          valueFrom:
            secretKeyRef:
              name: tado-passphrase
              key: passphrase
        volumeMounts:
        - name: tokens
          mountPath: /root/.tado-exporter
        livenessProbe:
          httpGet:
            path: /health
            port: 9100
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 9100
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
      volumes:
      - name: tokens
        persistentVolumeClaim:
          claimName: tado-tokens

---
# Service
apiVersion: v1
kind: Service
metadata:
  name: tado-exporter
  namespace: tado-monitoring
  labels:
    app: tado-exporter
spec:
  type: ClusterIP
  ports:
  - name: metrics
    port: 9100
    targetPort: 9100
  selector:
    app: tado-exporter

---
# ServiceMonitor (for Prometheus Operator)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: tado-exporter
  namespace: tado-monitoring
spec:
  selector:
    matchLabels:
      app: tado-exporter
  endpoints:
  - port: metrics
    interval: 60s
```

### Deploy to Kubernetes

```bash
# Create resources
kubectl apply -f tado-exporter-k8s.yaml

# Verify deployment
kubectl get pods -n tado-monitoring
kubectl get svc -n tado-monitoring

# Check logs
kubectl logs -n tado-monitoring deployment/tado-exporter

# Port forward for testing
kubectl port-forward -n tado-monitoring svc/tado-exporter 9100:9100

# Access metrics
curl http://localhost:9100/metrics
```

### Helm Chart (Future)

A Helm chart will be available for simplified deployment:

```bash
helm install tado-exporter ./helm/tado-prometheus-exporter \
  --set tadoPassphrase="your-passphrase" \
  --set homeId="12345"
```

---

## Systemd Service

For Linux systems with systemd:

### Create Service File

Create `/etc/systemd/system/tado-exporter.service`:

```ini
[Unit]
Description=Tado Prometheus Exporter
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=tado-exporter
Group=tado-exporter
WorkingDirectory=/var/lib/tado-exporter
ExecStart=/usr/local/bin/tado-prometheus-exporter \
  --token-path /var/lib/tado-exporter/token.json \
  --token-passphrase ${TADO_TOKEN_PASSPHRASE} \
  --port 9100 \
  --log-level info

# Environment file for secrets
EnvironmentFile=/etc/default/tado-exporter

# Auto-restart
Restart=on-failure
RestartSec=10

# Resource limits
MemoryLimit=512M
CPUQuota=50%

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=tado-exporter

[Install]
WantedBy=multi-user.target
```

### Setup

```bash
# Create user
sudo useradd --system --no-create-home tado-exporter

# Copy binary
sudo cp tado-exporter /usr/local/bin/tado-prometheus-exporter
sudo chmod +x /usr/local/bin/tado-prometheus-exporter

# Create data directory
sudo mkdir -p /var/lib/tado-exporter
sudo chown tado-exporter:tado-exporter /var/lib/tado-exporter
sudo chmod 700 /var/lib/tado-exporter

# Create environment file (with passphrase)
sudo cat > /etc/default/tado-exporter <<EOF
TADO_TOKEN_PASSPHRASE=your-secure-passphrase
EOF
sudo chmod 600 /etc/default/tado-exporter

# Reload systemd
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable tado-exporter
sudo systemctl start tado-exporter

# Check status
sudo systemctl status tado-exporter

# View logs
sudo journalctl -u tado-exporter -f
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
--home-id string           Optional home ID filter
--scrape-timeout int       API request timeout in seconds (default 10)
--log-level string         Log level: debug, info, warn, error (default info)
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
systemctl restart tado-exporter
```

### Logs Rotation

For standalone/systemd deployments, configure logrotate:

```bash
sudo tee /etc/logrotate.d/tado-exporter > /dev/null <<EOF
/var/log/tado-exporter.log {
  daily
  rotate 7
  compress
  delaycompress
  notifempty
  create 0640 tado-exporter tado-exporter
  postrotate
    systemctl reload tado-exporter >/dev/null 2>&1 || true
  endscript
}
EOF
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
curl -v https://my.tado.com/api/v2/homes -H "Authorization: Bearer <token>"
```

---

## Performance Tuning

### Resource Allocation

For different deployment sizes:

| Scale | CPU Request | Memory Request | Memory Limit |
|-------|-------------|-----------------|-------------|
| Small (1-5 zones) | 50m | 64Mi | 256Mi |
| Medium (6-20 zones) | 100m | 128Mi | 512Mi |
| Large (20+ zones) | 200m | 256Mi | 1Gi |

### Scrape Interval

- **Frequent updates**: 15-30 seconds
- **Standard**: 60 seconds (default)
- **Infrequent**: 5 minutes

Adjust in Prometheus config:

```yaml
scrape_configs:
  - job_name: 'tado-exporter'
    scrape_interval: 30s  # Change here
    static_configs:
      - targets: ['localhost:9100']
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
- [README.md](README.md) - Quick start
