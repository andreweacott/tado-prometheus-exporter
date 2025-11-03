# tado-prometheus-exporter

A Prometheus exporter for [Tado](https://www.tado.com/) heating systems, written in Go. Features OAuth 2.0 device code grant authentication with persistent token refresh—enabling unattended operation without user interaction.

## Features

- **OAuth 2.0 Device Code Grant**: Secure authentication without requiring user credentials to be stored
- **Persistent Token Storage**: Automatic token refresh for long-running, unattended operation
- **On-Demand Metrics**: Metrics fetched only when Prometheus scrapes the `/metrics` endpoint
- **Comprehensive Metrics**: Home-level and zone-level temperature, humidity, heating power, and window status
- **Docker Support**: Multi-stage build with minimal final image size
- **CI/CD Ready**: GitHub Actions workflows for testing and automated Docker image builds
- **Graceful Shutdown**: Proper signal handling for container orchestration

## Quick Start

### Prerequisites

- Go 1.23+ (for development)
- Docker (for containerized deployment)
- Tado API credentials (OAuth Client ID and Secret)

### Installation

#### Docker

```bash
docker run -v tado-tokens:/root/.tado-exporter \
  -p 9100:9100 \
  -e TADO_CLIENT_ID=<your_client_id> \
  -e TADO_CLIENT_SECRET=<your_client_secret> \
  ghcr.io/andreweacott/tado-prometheus-exporter:latest \
  --client-id=<your_client_id> \
  --client-secret=<your_client_secret>
```

#### From Source

```bash
git clone https://github.com/andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter
go build -o exporter ./cmd/exporter
./exporter --client-id=<your_client_id> --client-secret=<your_client_secret>
```

## Configuration

### Command-Line Flags

```
--client-id string
    OAuth 2.0 Client ID (required for initial authentication)

--client-secret string
    OAuth 2.0 Client Secret (required for initial authentication)

--port int
    HTTP server listen port (default: 9100)

--token-path string
    Path to token file for persistent storage (default: ~/.tado-exporter/token.json)

--home-id string
    Tado Home ID (optional, auto-detect if not provided)

--scrape-timeout int
    Maximum time in seconds to wait for API response (default: 10)

--log-level string
    Logging verbosity: debug, info, warn, error (default: info)
```

### Initial Authentication

On first run, if no valid token exists, the exporter will initiate the OAuth 2.0 device code flow:

1. A code will be displayed on the console
2. Visit the provided URL and authenticate with your Tado account
3. Enter the code when prompted
4. Token will be saved to the token file for future use

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
go build -o exporter ./cmd/exporter
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
    command:
      - --client-id=${TADO_CLIENT_ID}
      - --client-secret=${TADO_CLIENT_SECRET}
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

- **OAuth Tokens**: Stored in a protected file (`~/.tado-exporter/token.json`)
- **In Docker**: Use persistent volumes to protect token storage
- **Never commit**: OAuth credentials to version control
- **Environment Variables**: Use Docker secrets or environment variable injection in production

## Troubleshooting

### Authentication Issues

- Verify OAuth credentials are correct
- Check network connectivity to Tado API
- Ensure token file location is writable
- Review logs: `--log-level=debug`

### Metric Collection Issues

- Verify Home ID is correct (or omit to auto-detect)
- Check network connectivity
- Review scrape timeout settings
- Check Prometheus logs for scrape errors

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
