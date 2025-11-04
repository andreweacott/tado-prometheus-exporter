# HTTP Endpoints Documentation

This document describes the HTTP endpoints provided by the Tado Prometheus Exporter.

## Overview

The exporter exposes two main HTTP endpoints:
1. `/health` - Health check endpoint for container orchestration and monitoring
2. `/metrics` - Prometheus metrics endpoint for scraping

The server listens on a configurable port (default: 9100) and supports graceful shutdown.

## Server Configuration

### Startup

```bash
./exporter \
  --token-path=~/.tado-exporter/token.json \
  --token-passphrase=<your-secure-passphrase> \
  --port=9100 \
  --scrape-timeout=10
```

### Environment Variables

The exporter can also be configured via environment variables:
- `TADO_TOKEN_PATH` - Path to store encrypted token (default: ~/.tado-exporter/token.json)
- `TADO_TOKEN_PASSPHRASE` - Passphrase to encrypt/decrypt token (required)
- `TADO_PORT` - HTTP server port (default: 9100)
- `TADO_HOME_ID` - Optional: Filter to specific home
- `TADO_SCRAPE_TIMEOUT` - Metrics collection timeout in seconds (default: 10)

### First-Run Authentication

On first run, the exporter will perform OAuth device code authentication:

```
./exporter --token-path=~/.tado-exporter/token.json --token-passphrase=my-secret

No token found. Visit this link to authenticate:
https://my.tado.com/oauth/authorize?code=XXXX&device_code=YYYY

# After you authorize, the token is encrypted and saved to token path
Successfully authenticated. Token stored at: ~/.tado-exporter/token.json (encrypted with passphrase)
```

On subsequent runs, the existing token is loaded and reused. The token is automatically refreshed when needed.

## Endpoint Specifications

### GET `/health`

Health check endpoint for container orchestration systems (Kubernetes, Docker, etc.).

**Response Format:**
```json
{
  "status": "ok"
}
```

**Response Headers:**
- `Content-Type: application/json`

**HTTP Status Codes:**
- `200 OK` - Server is healthy and ready to serve metrics

**Example:**
```bash
curl http://localhost:9100/health
{"status":"ok"}
```

**Use Cases:**
- Kubernetes liveness/readiness probes
- Docker health checks
- Load balancer health verification
- Monitoring dashboards

---

### GET `/metrics`

Prometheus metrics endpoint for scraping metrics collected from the Tado API.

**Response Format:**
OpenMetrics text format (Prometheus exposition format)

**Response Headers:**
- `Content-Type: application/openmetrics-text; version=1.0.0; charset=utf-8`

**HTTP Status Codes:**
- `200 OK` - Metrics successfully collected and returned
- `500 Internal Server Error` - Error during metrics collection

**Timeout Behavior:**
- Each scrape is subject to the `--scrape-timeout` configuration
- If metrics collection exceeds timeout, metrics from the previous scrape are returned
- A warning is logged but the request succeeds with stale metrics

**Example:**
```bash
curl http://localhost:9100/metrics
# HELP tado_is_resident_present Whether anyone is home (1 = home, 0 = away)
# TYPE tado_is_resident_present gauge
tado_is_resident_present 1
# HELP tado_temperature_outside_celsius Outside temperature in Celsius
# TYPE tado_temperature_outside_celsius gauge
tado_temperature_outside_celsius 15.5
# HELP tado_temperature_measured_celsius Temperature measured in zone
# TYPE tado_temperature_measured_celsius gauge
tado_temperature_measured_celsius{home_id="123456",zone_id="1",zone_name="Living Room",zone_type="HEATING"} 21.3
...
```

**Available Metrics:**

#### Home-Level Metrics (no labels)

| Metric Name | Type | Description | Unit |
|---|---|---|---|
| `tado_is_resident_present` | Gauge | Whether anyone is home | 0 (away) or 1 (home) |
| `tado_solar_intensity_percentage` | Gauge | Solar radiation intensity | 0-100 (%) |
| `tado_temperature_outside_celsius` | Gauge | Outside temperature | Celsius |
| `tado_temperature_outside_fahrenheit` | Gauge | Outside temperature | Fahrenheit |

#### Zone-Level Metrics (labeled by home_id, zone_id, zone_name, zone_type)

| Metric Name | Type | Description | Unit |
|---|---|---|---|
| `tado_temperature_measured_celsius` | Gauge | Currently measured temperature | Celsius |
| `tado_temperature_measured_fahrenheit` | Gauge | Currently measured temperature | Fahrenheit |
| `tado_humidity_measured_percentage` | Gauge | Currently measured humidity | 0-100 (%) |
| `tado_temperature_set_celsius` | Gauge | Set temperature target | Celsius |
| `tado_temperature_set_fahrenheit` | Gauge | Set temperature target | Fahrenheit |
| `tado_heating_power_percentage` | Gauge | Heating power consumption | 0-100 (%) |
| `tado_is_window_open` | Gauge | Whether window is open | 0 (closed) or 1 (open) |
| `tado_is_zone_powered` | Gauge | Whether zone is powered on | 0 (off) or 1 (on) |

---

## HTTP Server Behavior

### Graceful Shutdown

The server responds to `SIGTERM` and `SIGINT` signals for graceful shutdown:

```bash
# Send SIGTERM to gracefully shutdown
kill -TERM <pid>
```

**Shutdown Process:**
1. Signal received and logged
2. Accept no new requests
3. Wait for in-flight requests to complete (max 10 seconds)
4. Close listener
5. Exit with status code 0

**Example Output:**
```
Received signal: terminated
Shutting down HTTP server...
HTTP server stopped
```

### Timeout Configuration

The `--scrape-timeout` parameter controls how long metrics collection is allowed to take:

```bash
./exporter --scrape-timeout=15
```

**Behavior:**
- Timeout applies per-scrape (per `/metrics` request)
- If Tado API is slow or unreachable, metrics collection times out
- Previous metrics are returned when timeout occurs
- Warning is logged to help diagnose issues

**Recommended Values:**
- Default: `10` seconds
- Fast networks: `5` seconds
- Slow/unreliable networks: `15-30` seconds

---

## Integration Examples

### Prometheus Configuration

Add to `prometheus.yml`:
```yaml
global:
  scrape_interval: 60s
  scrape_timeout: 10s

scrape_configs:
  - job_name: 'tado'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 60s
    scrape_timeout: 15s  # Should be longer than server's --scrape-timeout
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tado-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tado-exporter
  template:
    metadata:
      labels:
        app: tado-exporter
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9100"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: exporter
        image: tado-exporter:latest
        ports:
        - containerPort: 9100
        env:
        - name: TADO_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: tado-secrets
              key: client-id
        - name: TADO_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: tado-secrets
              key: client-secret
        - name: TADO_PORT
          value: "9100"
        livenessProbe:
          httpGet:
            path: /health
            port: 9100
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 9100
          initialDelaySeconds: 5
          periodSeconds: 10
```

### Docker Compose

```yaml
version: '3'
services:
  tado-exporter:
    image: tado-exporter:latest
    ports:
      - "9100:9100"
    environment:
      TADO_CLIENT_ID: ${TADO_CLIENT_ID}
      TADO_CLIENT_SECRET: ${TADO_CLIENT_SECRET}
      TADO_PORT: "9100"
      TADO_SCRAPE_TIMEOUT: "10"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9100/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

---

## Error Handling

### API Errors

When the Tado API returns an error or is unreachable:

**Behavior:**
- Warning is logged: `Warning: failed to collect Tado metrics: <error>`
- Previous/stale metrics are returned (if available)
- HTTP 200 response is still sent to Prometheus
- This allows graceful degradation if Tado service is temporarily unavailable

**Example Log:**
```
Warning: failed to collect Tado metrics: failed to fetch user: network timeout
```

### Timeout Errors

When metrics collection exceeds the scrape timeout:

**Behavior:**
- Collection is cancelled via context timeout
- Warning is logged
- Last known metrics are returned
- HTTP 200 response is sent

**Example Log:**
```
Warning: failed to collect Tado metrics: context deadline exceeded
```

---

## Testing Endpoints

### Using curl

```bash
# Test health endpoint
curl -v http://localhost:9100/health

# Test metrics endpoint
curl -s http://localhost:9100/metrics | head -20

# Test with custom timeout
curl --max-time 5 http://localhost:9100/metrics

# Watch metrics in real-time
watch -n 5 'curl -s http://localhost:9100/metrics | grep -v "^#"'
```

### Using Python requests

```python
import requests
import json

# Health check
resp = requests.get("http://localhost:9100/health")
print(json.dumps(resp.json(), indent=2))

# Metrics scrape
resp = requests.get("http://localhost:9100/metrics")
metrics = resp.text
print(metrics[:500])  # Print first 500 chars
```

### Using Go

```go
package main

import (
	"fmt"
	"net/http"
	"io"
)

func main() {
	// Health check
	resp, _ := http.Get("http://localhost:9100/health")
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	// Metrics
	resp, _ = http.Get("http://localhost:9100/metrics")
	body, _ = io.ReadAll(resp.Body)
	fmt.Println(string(body)[:500])
}
```

---

## Performance Considerations

### Request Latency

- `/health` endpoint: < 1ms (responds immediately)
- `/metrics` endpoint: Depends on Tado API response time (typically 1-5 seconds)
  - Network latency to Tado servers
  - API processing time
  - Timeout configuration

### Resource Usage

- Memory: ~50MB (Go binary + runtime)
- CPU: Minimal when idle, brief spike during metrics collection
- Network: ~5-10 requests per scrape to Tado API

### Optimization Tips

- Set `--scrape-timeout` appropriately for your network conditions
- Use Prometheus scrape cache to avoid collecting metrics too frequently
- Monitor exporter metrics to identify slow collections:
  ```promql
  # Metrics collection duration (if future version adds it)
  rate(tado_collection_duration_seconds[5m])
  ```

---

## Monitoring the Exporter

### Self-Metrics (Future Enhancement)

Future versions may expose self-metrics:
- `tado_exporter_up` - Whether exporter is running
- `tado_exporter_collection_duration_seconds` - Time to collect metrics
- `tado_exporter_api_errors_total` - Count of API errors
- `tado_exporter_last_collection_timestamp` - When metrics were last collected

---

## Troubleshooting

### Endpoints Not Responding

**Check if server is running:**
```bash
curl http://localhost:9100/health
```

**Check logs for startup errors:**
```bash
# If running via systemd
journalctl -u tado-exporter -n 50

# If running in Docker
docker logs <container-id>
```

### Metrics Not Updating

**Possible causes:**
1. Tado API authentication failed (check logs)
2. Network issues reaching Tado servers (check timeout)
3. Scrape interval too short (Prometheus scrapes faster than exporter can collect)

**Increase scrape timeout:**
```bash
./exporter --scrape-timeout=30
```

### High Memory Usage

**Possible causes:**
1. Memory leak in Prometheus library (file an issue)
2. Excessive number of zones/homes (creates many metric combinations)

**Verify with:**
```bash
ps aux | grep exporter
docker stats <container-id>
```

### Connection Refused

**Possible causes:**
1. Server hasn't started yet
2. Wrong port configuration
3. Port already in use

**Verify port:**
```bash
netstat -tlnp | grep 9100
lsof -i :9100
```

**Change port:**
```bash
./exporter --port=9101
```

---

## Security Considerations

### Network Access

- Do not expose `/metrics` endpoint to the public internet
- Use firewall rules to restrict access to trusted IPs
- Consider using a reverse proxy with authentication

### Token Security

- Token file permissions: `0600` (read/write by owner only)
- Store token file on encrypted filesystem
- Don't commit tokens to version control
- Rotate OAuth credentials regularly

### TLS/HTTPS

- Future versions may support TLS
- Currently only HTTP is supported
- Use a reverse proxy (nginx, etc.) for HTTPS if needed

---

## Changelog

### Phase 3 (Current)

- ✅ `/health` endpoint with JSON response
- ✅ `/metrics` endpoint with Prometheus format
- ✅ Graceful shutdown handling
- ✅ Configurable scrape timeout
- ✅ Error handling with stale metrics fallback
- ✅ Comprehensive integration tests
- ✅ This documentation

### Phase 4 (Upcoming)

- Actual metrics collection from Tado API
- Home-level metrics (resident presence, weather)
- Zone-level metrics (temperature, humidity, heating)
- Comprehensive error handling

### Phase 5 (Upcoming)

- TLS/HTTPS support
- Authentication/authorization
- Self-metrics for exporter monitoring
- Docker & CI/CD pipeline
