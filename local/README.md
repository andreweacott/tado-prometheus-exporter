# Local Testing Stack

⚠️ **IMPORTANT: This directory contains configuration for LOCAL DEVELOPMENT AND TESTING ONLY. Not suitable for production use.**

## Overview

This directory contains Docker Compose configuration and supporting files for running the complete Tado Prometheus Exporter stack locally with Prometheus and Grafana for development and testing.

## Files

- **`docker-compose.yml`** - Docker Compose configuration that orchestrates three services:
  - `exporter` - The Tado Prometheus exporter (built from the Dockerfile in the parent directory)
  - `prometheus` - Prometheus server for scraping and storing metrics
  - `grafana` - Grafana for visualizing metrics with pre-configured datasource and dashboard

- **`prometheus.yml`** - Prometheus configuration that defines:
  - Global scrape settings (15-second intervals)
  - Prometheus self-monitoring scrape config
  - Tado exporter scrape config (30-second intervals)

- **`grafana-provisioning-datasources.yml`** - Pre-configures Prometheus as a Grafana datasource
  - Automatically loaded on Grafana startup
  - Sets Prometheus as the default datasource
  - Configured to delete any duplicate "Prometheus" datasources on reload

- **`grafana-provisioning-dashboards.yml`** - Configures Grafana to load pre-built dashboards
  - Points to the dashboard directory for auto-loading

- **`tokens/`** - Local directory for storing encrypted Tado authentication tokens
  - Created automatically on first run
  - Persists across container restarts (see `.gitignore` - never committed to git)

- **`.gitignore`** - Prevents accidental commit of tokens and local volumes

Note: The example Tado Exporter dashboard is automatically imported from `../docs/examples/dashboards/tado-exporter.json`

## What's Pre-Configured

This stack is optimized for a smooth local testing experience:

✅ **Grafana Datasource** - Prometheus is automatically configured as a datasource
✅ **Grafana Dashboard** - The example Tado Exporter dashboard is auto-imported
✅ **Token Storage** - Correctly mounts to `/root/.tado-exporter` (exporter runs as root locally)
✅ **Network Connectivity** - All services are on the same Docker network with service-to-service DNS resolution

### Security Note

For local testing convenience, the exporter runs as **root** in the docker-compose stack. This is acceptable for development/testing but **not recommended for production**.

**Why root locally?** When running as root, `$HOME=/root` by default, so the exporter stores tokens at `/root/.tado-exporter` (the default location).

**For production**, use the standalone `docker run` method which properly runs the exporter as a non-root user (`exporter` UID 1000) with `$HOME=/home/exporter`.

## Quick Start

### Prerequisites

- Docker and Docker Compose
- A valid Tado account

### Running the Stack

```bash
# Navigate to this directory
cd local

# Start all services
TADO_TOKEN_PASSPHRASE=your-secure-passphrase docker-compose up -d

# View logs to get the authentication URL
docker-compose logs -f exporter

# Once authenticated, access the services:
# - Exporter metrics: http://localhost:9100/metrics
# - Exporter health: http://localhost:9100/health
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (credentials: admin/admin)
#   * Prometheus is pre-configured as a datasource
#   * The Tado Exporter dashboard is automatically imported and ready to use
```

### First Run Authentication

When running for the first time, the exporter will output an authentication URL:

```
Visit this URL to authenticate:
https://my.tado.com/oauth/authorize?code=XXXX&device_code=YYYY
```

1. Copy and visit that URL in your browser
2. Authorize the exporter with your Tado account
3. The exporter will automatically save the encrypted token

On subsequent runs, authentication is automatic.

## Stopping the Stack

```bash
cd local
docker-compose down

# Remove data volumes (if you want a fresh start)
docker-compose down -v
```

## Viewing Logs

```bash
cd local

# All services
docker-compose logs -f

# Specific service
docker-compose logs -f exporter
docker-compose logs -f prometheus
docker-compose logs -f grafana
```

## Customization

### Adjust Scrape Intervals

Edit `prometheus.yml` and modify the `scrape_interval` values:

```yaml
global:
  scrape_interval: 15s      # Change default interval

scrape_configs:
  - job_name: 'tado-exporter'
    scrape_interval: 30s    # Change exporter-specific interval
```

### Modify Exporter Configuration

Edit `docker-compose.yml` and update the `exporter` service environment variables:

```yaml
environment:
  TADO_TOKEN_PASSPHRASE: ${TADO_TOKEN_PASSPHRASE:-your_secure_passphrase}
  TADO_PORT: "9100"
  TADO_SCRAPE_TIMEOUT: "10"
  # Add other options as needed
  TADO_LOG_LEVEL: "debug"
  TADO_HOME_ID: "12345"
```

### Use Different Grafana Credentials

Edit `docker-compose.yml` and change the Grafana environment variables:

```yaml
environment:
  GF_SECURITY_ADMIN_PASSWORD: your-password
  GF_SECURITY_ADMIN_USER: your-username
```

## Troubleshooting

### Duplicate Datasources in Grafana

If you see duplicate "Prometheus" datasources:

```bash
# Option 1: Restart Grafana (provisioning will clean it up)
cd local && docker-compose restart grafana

# Option 2: Full stack restart
cd local && docker-compose down && docker-compose up -d

# Option 3: Remove Grafana volume and restart
cd local && docker-compose down -v && docker-compose up -d
```

The provisioning file is configured to delete any extra "Prometheus" datasources on load. If duplicates persist after restart, manually delete them via the Grafana UI (Configuration > Datasources > click the duplicate > Delete).

### Services won't start

```bash
# Check if ports are already in use
lsof -i :9100  # Exporter
lsof -i :9090  # Prometheus
lsof -i :3000  # Grafana

# Restart the stack
docker-compose restart
```

### Exporter not authenticating

```bash
# Check the logs
docker-compose logs exporter

# Remove the stored token and re-authenticate
docker exec tado-exporter rm /root/.tado-exporter/token.json
docker-compose restart exporter

# Check logs again for authentication URL
docker-compose logs -f exporter
```

### Prometheus not scraping metrics

1. Check Prometheus targets: http://localhost:9090/targets
2. Ensure the exporter container is healthy: `docker-compose ps`
3. Verify network connectivity: `docker network ls` and check the `monitoring` network

### Disk space issues

```bash
# Remove all data volumes and start fresh
docker-compose down -v
docker-compose up -d
```

## Data Persistence

The stack uses different storage strategies:

### Local Bind Mount (Host Filesystem)
- **`tokens/`** - Encrypted Tado authentication tokens
  - Stored on your host machine at `local/tokens/`
  - Persists across container restarts
  - Easy to inspect/backup/reset (just delete the directory)
  - Protected by `.gitignore` to prevent accidental commits

### Docker Named Volumes
- **`prometheus-data`** - Prometheus time-series data
- **`grafana-data`** - Grafana configurations and dashboards

These volumes are created automatically by Docker and managed independently of the local filesystem.

### Cleaning Up

```bash
# Stop containers (preserves all data)
docker-compose down

# Stop containers and remove named volumes (Prometheus and Grafana data lost, tokens preserved)
docker-compose down -v

# Delete tokens directory (remove encrypted tokens)
rm -rf tokens/

# Full clean reset
docker-compose down -v && rm -rf tokens/
```

## Networking

The three services communicate on a dedicated `monitoring` bridge network created by Docker Compose. Service-to-service hostnames:

- Exporter: `exporter:9100` (used by Prometheus)
- Prometheus: `prometheus:9090` (used by Grafana)
- Grafana: `grafana:3000`

## Next Steps

### For Production Deployment

- **Standalone Docker**: Use `docker run` with explicit environment variables and volume mounts
- **Kubernetes**: Deploy the exporter as a separate workload with your own Prometheus and Grafana
- **Docker Swarm**: Use `docker service create` with resource constraints and proper secret management

See the main [README.md](../README.md) and [ONBOARDING.md](../ONBOARDING.md) for detailed production deployment options.

### Integrating with Existing Prometheus/Grafana

If you already have Prometheus and Grafana running:

1. Only start the exporter:
   ```bash
   # Modify docker-compose.yml or run just the exporter service
   docker-compose up -d exporter
   ```

2. Add to your existing Prometheus configuration:
   ```yaml
   scrape_configs:
     - job_name: 'tado-exporter'
       static_configs:
         - targets: ['localhost:9100']
   ```

3. Import the Grafana dashboard from `docs/examples/dashboards/tado-exporter.json`

---

**For questions or issues, see the main repository documentation and issues tracker.**
