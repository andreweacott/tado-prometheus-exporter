# Tado Prometheus Exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/andreweacott/tado-prometheus-exporter)](https://goreportcard.com/report/github.com/andreweacott/tado-prometheus-exporter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-Available-blue?logo=docker)](https://hub.docker.com/r/andreweacott/tado-prometheus-exporter)
[![GitHub Release](https://img.shields.io/github/v/release/andreweacott/tado-prometheus-exporter)](https://github.com/andreweacott/tado-prometheus-exporter/releases)

Export [Tado](https://www.tado.com/) heating system metrics to Prometheus. Monitor your home's temperature, humidity, heating power, and occupancy with easy setup, encrypted token storage, and no OAuth app registration required.

## Why This Project?

Most Tado integrations require you to register an OAuth application upfront or provide permanent API credentials. **This exporter is different:**

- **No App Registration**: Uses OAuth 2.0 device code grant flow—authenticate directly like you would on a TV or smart device
- **Zero Configuration**: Just provide a passphrase; tokens are encrypted and stored locally
- **Truly Unattended**: One-time authentication, then run forever without interaction
- **Built for Homelabs**: Lightweight, designed to run on minimal hardware (Raspberry Pi, Docker, bare metal)
- **Prometheus Native**: Follows Prometheus best practices with proper instrumentation and error handling

### How It Compares

| Feature | This Exporter | Home Assistant | MQTT Bridge | Manual API |
|---------|---|---|---|---|
| **App Registration Required** | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes |
| **Encrypted Token Storage** | ✅ Yes | ❌ Plaintext | ❌ Plaintext | ❌ Manual |
| **Prometheus Compatible** | ✅ Yes | ❌ No | Partial | ❌ No |
| **No External Dependencies** | ✅ Yes | ❌ Full platform | ❌ Yes | N/A |
| **Docker Ready** | ✅ Yes | ✅ Yes | ✅ Yes | N/A |
| **Minimal Setup** | ✅ ~2 min | ❌ Complex | ❌ Complex | ❌ Manual |

## Features

✨ **Zero-Touch Setup**
- No OAuth app registration needed
- Device code authentication (scan QR code in browser)
- Automatic token refresh for months of unattended operation

🔐 **Security First**
- Tokens encrypted with your passphrase
- Encrypted storage at rest (`~/.tado-exporter/token.json`)
- Restrictive file permissions (owner read/write only)
- HTTPS-only API communication

📊 **Complete Metrics**
- Home-level: presence, outside temperature, solar intensity
- Per-zone: temperature (measured & set), humidity, heating power, window status, power state
- Exporter health: scrape duration, authentication status, error tracking

🚀 **Production Ready**
- Graceful shutdown with signal handling
- Partial collection (doesn't fail if one zone is offline)
- Configurable timeouts and retry logic
- Health check endpoint (`/health`)
- Structured logging (JSON or text)

🐳 **Container Native**
- Multi-stage Docker build (final image ~15MB)
- Docker Compose included
- Kubernetes compatible
- Health checks built-in

---

## Quick Start

### Option 1: Docker Compose (Recommended for Most Users)

```bash
# Clone the repository
git clone https://github.com/andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter

# Create .env file with your passphrase
cat > .env <<EOF
TADO_TOKEN_PASSPHRASE=your-secure-passphrase-here
EOF

# Start all services (exporter + Prometheus + Grafana)
docker-compose up -d

# Access services
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
# - Metrics: http://localhost:9100/metrics
```

**First run**: Visit the URL shown in `docker-compose logs exporter` to authorize with your Tado account.

### Option 2: Docker (Standalone)

```bash
# Build locally
docker build -t tado-exporter .

# Run container
docker run -d \
  --name tado-exporter \
  -p 9100:9100 \
  -v tado-tokens:/root/.tado-exporter \
  -e TADO_TOKEN_PASSPHRASE="your-secure-passphrase" \
  tado-exporter

# Check logs for authentication URL
docker logs tado-exporter
```

### Option 3: Standalone Binary

```bash
# Build from source
git clone https://github.com/andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter
go build -o tado-exporter ./cmd/exporter

# Run with your passphrase
./tado-exporter --token-passphrase="your-secure-passphrase"

# Follow the authentication prompt
```

---

## Configuration

### Common Options

```bash
./tado-exporter \
  --token-passphrase="your-passphrase" \           # Required
  --port=9100 \                                      # Metrics port (default: 9100)
  --scrape-timeout=10 \                             # API timeout seconds (default: 10)
  --home-id="12345" \                               # Optional: filter to specific home
  --log-level=info                                  # debug|info|warn|error (default: info)
```

### Environment Variables

All flags can be set via environment variables (useful for Docker):

```bash
export TADO_TOKEN_PASSPHRASE="your-passphrase"
export TADO_PORT=9100
export TADO_SCRAPE_TIMEOUT=10
export TADO_HOME_ID=12345
export TADO_LOG_LEVEL=info
```

---

## Authentication Flow

**First run** (one-time setup):

1. Start the exporter
2. You'll see:
   ```
   Visit this URL to authenticate:
   https://my.tado.com/authorize?user_code=ABCD-1234
   ```
3. Open the URL in your browser
4. Authorize the exporter with your Tado account
5. Token is encrypted and saved automatically

**Subsequent runs**:
- Exporter loads the encrypted token automatically
- Token is refreshed as needed
- No re-authentication required

---

## Prometheus Integration

### Add to `prometheus.yml`:

```yaml
global:
  scrape_interval: 60s

scrape_configs:
  - job_name: 'tado'
    static_configs:
      - targets: ['localhost:9100']
```

### Query Examples

```promql
# Current bedroom temperature
tado_temperature_measured_celsius{zone_name="Bedroom"}

# Average heating power across all zones
avg(tado_heating_power_percentage)

# Is anyone home?
tado_is_resident_present

# Temperature vs setpoint
tado_temperature_measured_celsius - tado_temperature_set_celsius
```

### Alerting

Example alert rules (in `prometheus.yml`):

```yaml
groups:
  - name: tado
    rules:
      # Alert if exporter is down
      - alert: TadoExporterDown
        expr: up{job="tado"} == 0
        for: 2m

      # Alert on high collection errors
      - alert: TadoHighErrorRate
        expr: rate(tado_exporter_scrape_errors_total[5m]) > 0.1
        for: 10m
```

See [`docs/alerts/README.md`](docs/alerts/README.md) for complete alert setup.

---

## Metrics Reference

### Home-Level Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tado_is_resident_present` | Gauge | Whether anyone is home (1=yes, 0=no) |
| `tado_solar_intensity_percentage` | Gauge | Solar radiation intensity (0-100%) |
| `tado_temperature_outside_celsius` | Gauge | Outside temperature (°C) |
| `tado_temperature_outside_fahrenheit` | Gauge | Outside temperature (°F) |

### Zone-Level Metrics

Labeled with: `home_id`, `zone_id`, `zone_name`, `zone_type`

| Metric | Type | Description |
|--------|------|-------------|
| `tado_temperature_measured_celsius` | Gauge | Current temperature (°C) |
| `tado_temperature_measured_fahrenheit` | Gauge | Current temperature (°F) |
| `tado_humidity_measured_percentage` | Gauge | Humidity (0-100%) |
| `tado_temperature_set_celsius` | Gauge | Target temperature (°C) |
| `tado_temperature_set_fahrenheit` | Gauge | Target temperature (°F) |
| `tado_heating_power_percentage` | Gauge | Heating output (0-100%) |
| `tado_is_window_open` | Gauge | Window open status (1=open, 0=closed) |
| `tado_is_zone_powered` | Gauge | Zone power state (1=on, 0=off) |

### Exporter Health Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tado_exporter_scrape_duration_seconds` | Histogram | Time to collect metrics (buckets: 0.1s, 0.2s, ..., 3.2s) |
| `tado_exporter_scrape_errors_total` | Counter | Total collection errors |
| `tado_exporter_authentication_valid` | Gauge | Is authentication valid? (1=yes, 0=no) |

---

## Documentation

| Document | Purpose |
|----------|---------|
| [**DEPLOYMENT.md**](docs/DEPLOYMENT.md) | Docker Compose, standalone, and Docker setup |
| [**TROUBLESHOOTING.md**](docs/TROUBLESHOOTING.md) | Common issues, diagnostics, and solutions |
| [**ARCHITECTURE.md**](docs/ARCHITECTURE.md) | Design decisions, library choices, and system design |
| [**HTTP_ENDPOINTS.md**](docs/HTTP_ENDPOINTS.md) | API reference and endpoint documentation |
| [**TADO_DEVICE_CODE_FLOW.md**](docs/TADO_DEVICE_CODE_FLOW.md) | OAuth 2.0 device code flow details |
| [**alerts/README.md**](docs/alerts/README.md) | Prometheus alerting setup and customization |

---

## Security

🔒 **Token Security**
- Encrypted at rest with AES-256 (via clambin/tado library)
- File permissions: `0600` (owner read/write only)
- Never logged or exposed in metrics
- Can be rotated by deleting the token file and re-authenticating

🔐 **Passphrase Guidelines**
- Use a strong passphrase (consider using a password manager)
- Store passphrase securely (Docker secrets, environment variables, not in git)
- Different passphrases for different deployments recommended

🌐 **Network Security**
- HTTPS-only communication with Tado API
- Consider using a reverse proxy with authentication if exposing `/metrics` externally
- Restrict access to the exporter port (default 9100) via firewall

---

## Development

### Prerequisites
- Go 1.25+
- Make (optional)
- golangci-lint (for linting)

### Build

```bash
go build -o tado-exporter ./cmd/exporter
```

### Test

```bash
go test -v -race ./...
```

### Lint

```bash
golangci-lint run ./...
```

### Docker Build

```bash
docker build -t tado-prometheus-exporter:dev .
```

---

## Troubleshooting

### Common Issues

**Q: "Device code expired"**
- You have 5 minutes to complete authentication
- Check internet connectivity and try again

**Q: "Token file corrupted or invalid"**
- Verify passphrase is correct
- Check file permissions: `ls -la ~/.tado-exporter/token.json`
- Delete and re-authenticate: `rm ~/.tado-exporter/token.json && docker restart tado-exporter`

**Q: "No metrics returned"**
- Check exporter is running: `curl http://localhost:9100/health`
- Check logs: `docker logs tado-exporter`
- Increase timeout if your network is slow: `--scrape-timeout=30`

**Q: "Prometheus not scraping metrics"**
- Verify Prometheus config has exporter in scrape_configs
- Check Prometheus targets page: http://localhost:9090/targets
- Ensure exporter port (9100) is accessible from Prometheus

👉 **For more help**: See [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

---

## Examples

### Grafana Dashboard

After setting up Prometheus as a data source:

1. Create a new dashboard
2. Add panels querying Tado metrics:
   - `tado_temperature_measured_celsius` (temperature graph)
   - `tado_heating_power_percentage` (heating status)
   - `tado_is_resident_present` (occupancy indicator)

### Temperature Alerting

Alert when temperature drops below target:

```yaml
- alert: TadoTemperatureLow
  expr: (tado_temperature_measured_celsius - tado_temperature_set_celsius) < -2
  for: 15m
```

### Energy Monitoring

Track heating patterns:

```promql
# Average heating power per zone
avg by (zone_name) (tado_heating_power_percentage)

# Temperature efficiency (delta from setpoint)
avg(tado_temperature_measured_celsius - tado_temperature_set_celsius)
```

---

## Contributing

We welcome contributions! Please follow these steps:

1. **Fork** the repository
2. **Create a feature branch**: `git checkout -b feature/my-feature`
3. **Make your changes** and write tests
4. **Run tests and linting**:
   ```bash
   go test -v ./...
   golangci-lint run ./...
   ```
5. **Commit with clear messages**
6. **Push to your fork** and **open a Pull Request**

### Development Guidelines

- Follow Go conventions and idioms
- Keep functions small and focused
- Add tests for new features
- Update documentation as needed
- One feature/fix per PR

---

## License

Licensed under the MIT License. See [LICENSE](LICENSE) file for details.

---

## Support & Community

- 📖 **Documentation**: See [docs/](docs/) for detailed guides
- 🐛 **Issues**: [Report bugs or request features](https://github.com/andreweacott/tado-prometheus-exporter/issues)
- 💬 **Discussions**: [Ask questions or share ideas](https://github.com/andreweacott/tado-prometheus-exporter/discussions)
- ⭐ **Like it?** Consider starring the repository to show your support

---

## Related Projects

- [clambin/tado](https://github.com/clambin/tado) - Tado API Go library
- [Prometheus](https://prometheus.io/) - Monitoring and alerting toolkit
- [Grafana](https://grafana.com/) - Visualization platform

---

## Acknowledgments

- Built with [clambin/tado](https://github.com/clambin/tado) Tado API library
- Follows [Prometheus exporter best practices](https://prometheus.io/docs/practices/instrumentation/)
- Inspired by the homelab community

---

**Made with ❤️ for home automation enthusiasts and homelabs**
