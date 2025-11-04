# tado-prometheus-exporter

A Prometheus exporter for [Tado](https://www.tado.com/) heating systems, written in Go. Features OAuth 2.0 device code grant authentication with encrypted token storage—enabling unattended operation without user interaction, with no upfront OAuth app registration required.

## Features

- **OAuth 2.0 Device Code Grant**: Automatic, zero-config authentication with encrypted token storage
- **No App Registration Required**: Uses device code OAuth flow—just provide a passphrase for token encryption
- **Encrypted Token Storage**: Tokens stored securely on disk with user-provided passphrase
- **Automatic Token Refresh**: Long-running unattended operation without user interaction
- **On-Demand Metrics**: Metrics fetched only when Prometheus scrapes the `/metrics` endpoint
- **Comprehensive Metrics**: Home-level and zone-level temperature, humidity, heating power, and window status
- **Docker Support**: Multi-stage build with minimal final image size
- **CI/CD Ready**: GitHub Actions workflows for testing and automated Docker image builds
- **Graceful Shutdown**: Proper signal handling for container orchestration

## Quick Start

### Prerequisites

- Go 1.23+ (for development)
- Docker (for containerized deployment)
- A secure passphrase for token encryption (any string you choose)

### Installation

#### Docker

```bash
docker run -v tado-tokens:/root/.tado-exporter \
  -p 9100:9100 \
  -e TADO_TOKEN_PASSPHRASE=<your_secure_passphrase> \
  ghcr.io/andreweacott/tado-prometheus-exporter:latest \
  --token-passphrase=<your_secure_passphrase>
```

On first run, you'll be prompted to authenticate with your Tado account via device code flow (visit the provided URL in your browser).

#### From Source

```bash
git clone https://github.com/andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter
go build -o tado-exporter ./cmd/exporter
./tado-exporter --token-passphrase=<your_secure_passphrase>
```

On first run, the exporter will guide you through device code authentication. After that, it will reuse the encrypted token for subsequent runs.

## Configuration

### Command-Line Flags

```
--token-passphrase string
    Passphrase to encrypt/decrypt Tado token (required)

--token-path string
    Path to encrypted token file (default: ~/.tado-exporter/token.json)

--port int
    HTTP server listen port (default: 9100)

--home-id string
    Tado Home ID (optional, auto-detect if not provided)

--scrape-timeout int
    Maximum time in seconds to wait for metrics collection (default: 10)

--log-level string
    Logging verbosity: debug, info, warn, error (default: info)
```

### Environment Variables

All configuration can be set via environment variables:

```
TADO_TOKEN_PASSPHRASE      # Token passphrase (required)
TADO_TOKEN_PATH            # Path to token file
TADO_PORT                  # HTTP server port
TADO_HOME_ID               # Tado home ID
TADO_SCRAPE_TIMEOUT        # Metrics collection timeout
```

### First-Run Authentication

On first run, the exporter will automatically initiate OAuth 2.0 device code authentication:

1. You'll see a message with a verification URL and code
2. Visit the URL in your web browser
3. Authorize the application with your Tado account
4. Token will be encrypted with your passphrase and saved for future use

**Example:**
```
No token found. Visit this link to authenticate:
https://my.tado.com/oauth/authorize?code=XXXX&device_code=YYYY

Successfully authenticated. Token stored at: ~/.tado-exporter/token.json (encrypted with passphrase)
```

On subsequent runs, the exporter loads the encrypted token automatically (no re-authentication needed).

## Endpoints

- `GET /metrics` - Prometheus metrics endpoint
- `GET /health` - Health check endpoint

## Metrics

### Home-Level Metrics

- `tado_is_resident_present` - Whether anyone is home (0/1)
- `tado_solar_intensity_percentage` - Solar radiation intensity (0-100%)
- `tado_temperature_outside_celsius` - Outside temperature in Celsius
- `tado_temperature_outside_fahrenheit` - Outside temperature in Fahrenheit

### Zone-Level Metrics

All zone metrics are labeled with `zone_id`, `zone_name`, and `zone_type`.

- `tado_temperature_measured_celsius` - Measured temperature in Celsius
- `tado_temperature_measured_fahrenheit` - Measured temperature in Fahrenheit
- `tado_humidity_measured_percentage` - Measured humidity (0-100%)
- `tado_temperature_set_celsius` - Set/target temperature in Celsius
- `tado_temperature_set_fahrenheit` - Set/target temperature in Fahrenheit
- `tado_heating_power_percentage` - Heating power (0-100%)
- `tado_is_window_open` - Window open status (0/1)
- `tado_is_zone_powered` - Zone power status (0/1)

## Development

### Build

```bash
go build -o tado-exporter ./cmd/exporter
```

### Test

```bash
go test -v ./...
```

### Lint

```bash
golangci-lint run ./...
```

### Docker Build

```bash
docker build -t tado-prometheus-exporter .
```

## Deployment

### Docker Compose

```yaml
version: '3.8'

services:
  exporter:
    build: .
    ports:
      - "9100:9100"
    volumes:
      - tado-tokens:/root/.tado-exporter
    environment:
      TADO_TOKEN_PASSPHRASE: ${TADO_TOKEN_PASSPHRASE}
    command:
      - --token-passphrase=${TADO_TOKEN_PASSPHRASE}
      - --port=9100

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'

volumes:
  tado-tokens:
```

Create a `.env` file with your token passphrase:
```bash
TADO_TOKEN_PASSPHRASE=your_secure_passphrase_here
```

Then run:
```bash
docker-compose up
```

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'tado'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 5m  # Adjust as needed
```

## Security

- **Encrypted Tokens**: Tokens are encrypted with your passphrase and stored in `~/.tado-exporter/token.json`
- **File Permissions**: Token file is created with restrictive permissions (owner read/write only)
- **Passphrase Protection**: Choose a strong passphrase—it's the key to your encrypted token
- **In Docker**: Use Docker secrets or `.env` files (not committed to git) for passphrase management
- **Environment Variables**: Pass `TADO_TOKEN_PASSPHRASE` via Docker secrets, not in compose files
- **Never commit**: Token files or passphrases to version control
- **Network**: The exporter communicates with Tado API over HTTPS. Consider using a reverse proxy with authentication if exposing metrics externally.

## Troubleshooting

### Authentication Issues

- **No token prompt on first run**: Check that you're running the exporter with `--token-passphrase` and internet connectivity
- **"Failed to create OAuth2 client"**: Verify your passphrase is correct and token file location is writable
- **Token file permission denied**: Ensure the directory `~/.tado-exporter/` has correct permissions
- **Device code expired**: You have 5 minutes to complete authentication. Check your network connection and try again
- **Review logs**: Run with `--log-level=debug` for detailed diagnostics

### Metric Collection Issues

- **No metrics returned**: Verify Home ID is correct (or omit to auto-detect)
- **Timeout errors**: Increase `--scrape-timeout` if your network is slow
- **Connection refused**: Ensure the exporter port (default 9100) is not in use
- **Check Prometheus**: Review Prometheus logs for scrape errors: `curl http://localhost:9100/metrics`

### Docker Issues

- **Container exits immediately**: Check logs with `docker logs <container-id>`
- **Token file not persisting**: Ensure volume is mounted correctly: `-v tado-tokens:/root/.tado-exporter`
- **Passphrase not passed correctly**: Use environment variables or `.env` files, not hardcoded in compose files

### Additional Help

See [HTTP_ENDPOINTS.md](HTTP_ENDPOINTS.md) for detailed endpoint documentation.

## Architecture

See [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for detailed architecture and development phases.

## License

[Add your license here]

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## Support

For issues, questions, or suggestions, please open an issue on GitHub.
