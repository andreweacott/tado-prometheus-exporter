# Tado Prometheus Exporter - Architecture & Design Decisions

## Overview

This document explains the architectural decisions and design choices made in the Tado Prometheus Exporter project.

---

## Library Choices

### Primary Tado Library: `clambin/tado/v2`

**Choice**: Use `github.com/clambin/tado/v2` as the primary Tado API client library

**Rationale**:
- **Authentication**: Native support for OAuth2 device code flow with encrypted token storage
- **Token Management**: Automatic token refresh without requiring custom implementation
- **API Coverage**: Complete support for Tado API endpoints needed for metrics collection (homes, zones, weather)
- **Code Generation**: Generated from OpenAPI spec, ensuring compatibility with official API
- **Maintenance**: Active development and community support
- **Type Safety**: Strongly typed Go structs for API responses

### Direct Dependencies

| Library | Version | Purpose |
|---------|---------|---------|
| `clambin/tado/v2` | v2.6.2 | Tado API client |
| `prometheus/client_golang` | v1.23.2 | Prometheus metrics |
| `sirupsen/logrus` | v1.9.3 | Structured logging |
| `stretchr/testify` | v1.11.1 | Testing utilities |
| `golang.org/x/oauth2` | v0.32.0 | OAuth2 client (required by clambin/tado) |

### Minimalist Approach

- Kept dependencies minimal to reduce surface area and attack vectors
- All transitive dependencies are indirect (managed by direct dependency libraries)
- Regular `go mod tidy` to remove any unused dependencies

---

## Package Structure

```
pkg/
├── auth/           # OAuth2 authentication and Tado client initialization
├── collector/      # Prometheus collector for Tado metrics
├── config/         # Configuration management (flags, env vars, validation)
├── logger/         # Structured logging with logrus
└── metrics/        # Prometheus metric definitions and exporter health metrics

cmd/
└── exporter/       # Main application (server, HTTP handlers, graceful shutdown)
```

### Package Responsibilities

#### `pkg/auth`
- OAuth2 device code authentication flow
- Token storage with encryption
- Authenticated Tado client creation

#### `pkg/collector`
- Prometheus collector implementation
- Fetches metrics from Tado API
- Handles partial collection (graceful degradation on errors)
- Emits both Tado metrics and exporter health metrics

#### `pkg/config`
- Flag parsing with environment variable override support
- Configuration validation
- Precedence: CLI flags > env vars > defaults

#### `pkg/logger`
- Structured logging with logrus
- JSON and text output formats
- Log levels: debug, info, warn, error

#### `pkg/metrics`
- Tado metrics descriptors (temperature, humidity, etc.)
- Exporter health metrics (scrape duration, errors, auth status)
- Metric registration with Prometheus

#### `cmd/exporter`
- Server startup and graceful shutdown
- HTTP handler registration
- Metrics endpoint setup
- Health check endpoint

---

## Key Design Patterns

### 1. Graceful Error Handling

**Pattern**: Partial Collection on Failure

Instead of failing completely when one API call fails, the system collects what it can:

```go
// Collect home metrics, then zone metrics
// If home metrics fail, still collect zone metrics
// If individual zones fail, collect others
```

**Benefit**: Operators get partial metrics even during API issues, enabling better diagnostics

### 2. Configuration Precedence

Pattern: CLI flags > environment variables > defaults

```
Precedence Order:
1. CLI flag provided    → use CLI value
2. Env var set         → use env value
3. Neither            → use default
```

**Benefit**: Supports multiple deployment scenarios (Docker, standalone, CLI)

### 3. Metrics Validation

All metrics extracted from Tado API responses are validated against known valid ranges before recording to Prometheus.

**Validation Ranges**:

| Metric | Min | Max | Unit | Rationale |
|--------|-----|-----|------|-----------|
| Temperature (Celsius) | -50 | 60 | °C | Typical building range |
| Temperature (Fahrenheit) | -58 | 140 | °F | Converted from Celsius |
| Humidity | 0 | 100 | % | Relative humidity definition |
| Heating Power | 0 | 100 | % | Percentage output range |

**Benefit**: Prevents invalid data from corrupting Prometheus time-series

### 4. Test Isolation

Pattern: Use custom Prometheus registries in tests to avoid conflicts

```go
registry := prometheus.NewRegistry()  // Local registry for test
registry.Register(metric)
```

**Benefit**: Tests can run in parallel without registration conflicts

---

## Error Handling Strategy

### Levels of Error Handling

| Level | Strategy | Example |
|-------|----------|---------|
| **Fatal** | Exit immediately | Invalid config, can't start server |
| **Critical** | Fail operation, log + alert | Can't authenticate with Tado |
| **Degraded** | Continue with partial results | One zone fails, collect other zones |
| **Info** | Log for diagnostics | Retry after temporary network error |

### Metrics for Error Visibility

```
tado_exporter_scrape_errors_total       # Track collection failures
tado_exporter_authentication_valid      # Alert on auth issues
tado_exporter_authentication_errors_total # Track auth failure trends
tado_exporter_last_authentication_success_unix  # Detect stale auth
```

---

## Observability

### Exporter Health Metrics

**Scrape Performance**:
- `tado_exporter_scrape_duration_seconds` (histogram)
- Buckets: 0.1s, 0.2s, 0.4s, 0.8s, 1.6s, 3.2s

**Error Tracking**:
- `tado_exporter_scrape_errors_total` (counter)

**Authentication Health**:
- `tado_exporter_authentication_valid` (gauge: 1=valid, 0=invalid)
- `tado_exporter_authentication_errors_total` (counter)
- `tado_exporter_last_authentication_success_unix` (gauge: timestamp)

**Build Information**:
- `tado_exporter_build_info` (gauge: always 1)

### Structured Logging

All log output includes context fields:
- `home_id`: Home identifier
- `zone_id`: Zone identifier
- `zone_name`: Zone name
- `error`: Error message
- `timestamp`: ISO 8601 format

### Prometheus Alert Examples

```yaml
# Alert when authentication is invalid
- alert: TadoAuthenticationInvalid
  expr: tado_exporter_authentication_valid == 0
  for: 5m

# Alert on scrape failures
- alert: TadoScrapeErrors
  expr: rate(tado_exporter_scrape_errors_total[5m]) > 0.1
```

---

## Security Considerations

### Token Storage

- Tokens stored encrypted with passphrase (via clambin/tado library)
- Passphrase must be provided as environment variable or CLI flag
- Never logged or exposed in metrics

### Configuration Security

- Secrets not logged by default (only via DEBUG level)
- Environment variables used for sensitive config
- CLI flags override for testing without persistent secrets

### Least Privilege

- Only requests necessary Tado API endpoints
- No admin credentials required (user-level API)
- OAuth2 scopes limited to required permissions

---

## Performance Considerations

### Metrics Collection

- Single goroutine for simplicity (no concurrent API calls)
- Context timeout prevents hanging requests
- Partial collection avoids retrying failed APIs indefinitely

### Memory Usage

- Metrics accumulated in gauges/counters (minimal memory)
- Single collection cycle before export
- No in-memory buffering of raw API responses

### Network Efficiency

- Batches zone state requests (single API call for all zones)
- Reuses authenticated HTTP client connection
- Configurable scrape timeout (default 10s)

---

## Testing Strategy

### Test Coverage

| Package | Type | Count |
|---------|------|-------|
| `auth` | Unit | 3 |
| `collector` | Unit | 10+ |
| `config` | Unit | 15+ |
| `logger` | Unit | 20+ |
| `metrics` | Unit | 13 |
| `exporter` | Integration | 11 |
| **Total** | | **80+** |

### Test Isolation

- Each test uses isolated Prometheus registry
- No shared state between tests
- Tests run in parallel
- Environment variables isolated per test

---

## Future Enhancement Opportunities

### Could Add

1. **Circuit Breaker**: Skip API calls during outages, cache last-known metrics
2. **Retry with Exponential Backoff**: Automatic retry for transient errors
3. **Multi-Home Support**: Collect metrics from multiple homes in parallel
4. **Custom Metrics**: Allow users to add custom metric collection

### Won't Add

- Complex caching strategies (Prometheus handles this)
- Real-time metric streaming (Prometheus is polling-based)
- Metric aggregation (use Prometheus queries instead)

---

## References

- [Tado API Documentation](https://support.tado.com/hc/en-us/articles/8113175915041)
- [Prometheus Client Libraries](https://prometheus.io/docs/instrumenting/clientlibs/)
- [clambin/tado GitHub](https://github.com/clambin/tado)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/instrumentation/)
