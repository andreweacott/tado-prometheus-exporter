# HTTP Endpoints Documentation

This document describes the HTTP endpoints provided by the Tado Prometheus Exporter.

## Overview

The exporter exposes two main HTTP endpoints:
1. `/health` - Health check endpoint for container orchestration and monitoring
2. `/metrics` - Prometheus metrics endpoint for scraping

The server listens on a configurable port (default: 9100).

## Configuration

### Server Startup

```bash
./exporter \
  --token-path=~/.tado-exporter/token.json \
  --token-passphrase=<your-secure-passphrase> \
  --port=9100 \
  --scrape-timeout=10
```

### Environment Variables

- `TADO_TOKEN_PATH` - Path to store encrypted token (default: ~/.tado-exporter/token.json)
- `TADO_TOKEN_PASSPHRASE` - Passphrase to encrypt/decrypt token (required)
- `TADO_PORT` - HTTP server port (default: 9100)
- `TADO_HOME_ID` - Optional: Filter to specific home
- `TADO_SCRAPE_TIMEOUT` - Metrics collection timeout in seconds (default: 10)

---

## Endpoints

### GET `/health`

Health check endpoint for container orchestration systems (Kubernetes, Docker, etc.).

**Response:**
```json
{
  "status": "ok"
}
```

**Status Codes:**
- `200 OK` - Server is healthy and ready to serve metrics

**Example:**
```bash
curl http://localhost:9100/health
```

---

### GET `/metrics`

Prometheus metrics endpoint for scraping metrics collected from the Tado API.

**Response Format:** OpenMetrics text format (Prometheus exposition format)

**Status Codes:**
- `200 OK` - Metrics successfully collected and returned
- `500 Internal Server Error` - Error during metrics collection

**Example:**
```bash
curl http://localhost:9100/metrics
```

**Timeout Behavior:**
- Each scrape is subject to the `--scrape-timeout` configuration
- If metrics collection exceeds timeout, metrics from the previous scrape are returned
- A warning is logged but the request succeeds with stale metrics

## Available Metrics

### Home-Level Metrics (no labels)

| Metric Name | Type | Description | Unit |
|---|---|---|---|
| `tado_is_resident_present` | Gauge | Whether anyone is home | 0 (away) or 1 (home) |
| `tado_solar_intensity_percentage` | Gauge | Solar radiation intensity | 0-100 (%) |
| `tado_temperature_outside_celsius` | Gauge | Outside temperature | Celsius |
| `tado_temperature_outside_fahrenheit` | Gauge | Outside temperature | Fahrenheit |

### Zone-Level Metrics

Labeled by: `home_id`, `zone_id`, `zone_name`, `zone_type`

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

### Example Metrics Output

```bash
curl http://localhost:9100/metrics | head -30

# HELP tado_is_resident_present Whether anyone is home (1 = home, 0 = away)
# TYPE tado_is_resident_present gauge
tado_is_resident_present 1

# HELP tado_temperature_outside_celsius Outside temperature in Celsius
# TYPE tado_temperature_outside_celsius gauge
tado_temperature_outside_celsius 15.5

# HELP tado_temperature_measured_celsius Temperature measured in zone
# TYPE tado_temperature_measured_celsius gauge
tado_temperature_measured_celsius{home_id="123456",zone_id="1",zone_name="Living Room",zone_type="HEATING"} 21.3
```

---

## Server Behavior

### Graceful Shutdown

The server responds to `SIGTERM` and `SIGINT` signals:

```bash
kill -TERM <pid>
```

**Shutdown Process:**
1. Signal received and logged
2. Accept no new requests
3. Wait for in-flight requests to complete (max 10 seconds)
4. Close listener
5. Exit with status code 0

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

## Integration with Prometheus

### Basic Configuration

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
    scrape_timeout: 15s
```

### Query Examples

```promql
# Current temperature in bedroom
tado_temperature_measured_celsius{zone_name="Bedroom"}

# Average heating power
avg(tado_heating_power_percentage)

# Temperature difference from setpoint
tado_temperature_measured_celsius - tado_temperature_set_celsius

# Is anyone home?
tado_is_resident_present
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

### Timeout Errors

When metrics collection exceeds the scrape timeout:

**Behavior:**
- Collection is cancelled via context timeout
- Warning is logged
- Last known metrics are returned
- HTTP 200 response is sent

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
```

### Using Python

```python
import requests

# Health check
resp = requests.get("http://localhost:9100/health")
print(resp.json())

# Metrics scrape
resp = requests.get("http://localhost:9100/metrics")
print(resp.text[:500])
```

---

## Performance Considerations

### Request Latency

- `/health` endpoint: < 1ms (responds immediately)
- `/metrics` endpoint: Depends on Tado API response time (typically 1-5 seconds)

### Resource Usage

- Memory: ~50MB (Go binary + runtime)
- CPU: Minimal when idle, brief spike during metrics collection
- Network: ~5-10 requests per scrape to Tado API

---

## Troubleshooting

### Endpoints Not Responding

**Check if server is running:**
```bash
curl http://localhost:9100/health
```

**Check logs:**
```bash
docker logs tado-exporter
# or
journalctl -u tado-exporter -n 50
```

### Metrics Not Updating

**Possible causes:**
1. Tado API authentication failed
2. Network issues reaching Tado servers
3. Scrape interval too short

**Increase scrape timeout:**
```bash
./exporter --scrape-timeout=30
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
- Consider using a reverse proxy with authentication if exposing externally

### Token Security

- Token file permissions: `0600` (read/write by owner only)
- Store token file on encrypted filesystem
- Don't commit tokens to version control

---

## Related Documentation

- [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment options
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
