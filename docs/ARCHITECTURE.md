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

**Alternative Considered**: `gonzolino/gotado/v2`
- **Status**: REMOVED (replaced by clambin/tado)
- **Reason**: Removed to reduce dependency footprint. All enums and types were moved to string/bool comparisons since tests only needed simple conversions, not actual library functionality

---

## Dependency Management

### Direct Dependencies

| Library | Version | Purpose |
|---------|---------|---------|
| `clambin/tado/v2` | v2.6.2 | Tado API client |
| `prometheus/client_golang` | v1.23.2 | Prometheus metrics |
| `sirupsen/logrus` | v1.9.3 | Structured logging |
| `stretchr/testify` | v1.11.1 | Testing utilities |
| `golang.org/x/oauth2` | v0.32.0 | OAuth2 client (required by clambin/tado) |

### Removed Dependencies

- **gonzolino/gotado/v2**: Removed as part of P3.1 cleanup (unused after test refactoring)

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
- Files: `authenticator.go` (core implementation)

#### `pkg/collector`
- Prometheus collector implementation
- Fetches metrics from Tado API
- Handles partial collection (graceful degradation on errors)
- Emits both Tado metrics and exporter health metrics
- Files: `collector.go` (main), `collector_test.go` (tests)

#### `pkg/config`
- Flag parsing with environment variable override support
- Configuration validation
- Precedence: CLI flags > env vars > defaults
- Files: `config.go` (implementation), `config_test.go` (15+ tests)

#### `pkg/logger`
- Structured logging with logrus
- JSON and text output formats
- Log levels: debug, info, warn, error
- Context field helpers (home_id, zone_id, request_id, etc.)
- Files: `logger.go` (implementation), `logger_test.go` (20+ tests)

#### `pkg/metrics`
- Tado metrics descriptors (temperature, humidity, etc.)
- Exporter health metrics (scrape duration, errors, auth status)
- Metric registration with Prometheus
- Files: `metrics.go` (Tado metrics), `exporter_metrics.go` (health metrics)

#### `cmd/exporter`
- Server startup and graceful shutdown
- HTTP handler registration
- Metrics endpoint setup
- Health check endpoint
- Files: `main.go`, `server.go`, `server_test.go`

---

## Key Design Patterns

### 1. Graceful Error Handling (P2.2)

**Pattern**: Partial Collection on Failure

Instead of failing completely when one API call fails, the system collects what it can:

```go
// Before: Early return stops all collection
if err := collectHomeMetrics() {
    return err  // ❌ No metrics exported
}

// After: Continue collection, track errors
if err := collectHomeMetrics() {
    logError(err)
    errors = append(errors, err)
    // Continue to next metric
}
```

**Benefit**: Operators get partial metrics even during API issues, enabling better diagnostics

### 2. Optional Feature Attachment

Pattern: Components can optionally accept monitoring features without breaking:

```go
collector := NewTadoCollector(client, metrics, timeout, homeID)
collector.WithExporterMetrics(exporterMetrics)  // Optional monitoring
```

**Benefit**: Metrics are optional; core functionality works without them

### 3. Configuration Precedence

Pattern: CLI flags > environment variables > defaults

```
Precedence Order:
1. CLI flag provided    → use CLI value
2. Env var set         → use env value
3. Neither            → use default
```

**Benefit**: Supports multiple deployment scenarios (Docker, Kubernetes, CLI)

### 4. Test Isolation

Pattern: Use custom Prometheus registries in tests to avoid conflicts

```go
registry := prometheus.NewRegistry()  // ✅ Local registry for test
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

## Metrics Validation Strategy (P2.3)

### Validation Approach

All metrics extracted from Tado API responses are validated against known valid ranges before recording to Prometheus. Invalid metrics are skipped with warnings logged.

### Validation Ranges

| Metric | Type | Min | Max | Unit | Rationale |
|--------|------|-----|-----|------|-----------|
| Temperature (Celsius) | float32 | -50 | 60 | °C | Typical building range |
| Temperature (Fahrenheit) | float32 | -58 | 140 | °F | Converted from Celsius |
| Humidity | float32 | 0 | 100 | % | Relative humidity definition |
| Heating Power | float32 | 0 | 100 | % | Percentage output range |

### Implementation

**Extraction Helpers** (`pkg/collector/zone_metrics.go`):
```go
func extractZoneTemperature(zoneState *tado.ZoneState) (*float32, *float32)
func extractZoneHumidity(zoneState *tado.ZoneState) *float32
func extractHeatingPower(zoneState *tado.ZoneState) *float32
```

**Validation Functions**:
```go
func validateTemperature(temp float32, fieldName string) error
func validateHumidity(humidity float32, fieldName string) error
func validatePower(power float32, fieldName string) error
func ValidateZoneMetrics(metrics *ZoneMetrics) []error
```

**Integration in Collection** (`pkg/collector/collector.go`):
```go
// Extract metrics
metrics := ExtractAllZoneMetrics(&zoneState)

// Validate metrics
validationErrors := ValidateZoneMetrics(metrics)
for _, err := range validationErrors {
    log.Warn("Validation failed", "error", err.Error())
}

// Record only valid metrics
if err := validateTemperature(temp, "measured"); err != nil {
    log.Warn("Skipping invalid temperature", "value", temp)
} else {
    recordMetric(temp)
}
```

### Error Handling

- **Invalid values are skipped** (not recorded to Prometheus)
- **Warnings logged** with context (home_id, zone_id, field_name, value, reason)
- **Collection continues** for other metrics even if one fails (graceful degradation)
- **No metrics available** → no gauge update (Prometheus uses last-known value)

### Benefits

1. **Data Quality**: Prevents invalid data from corrupting Prometheus time-series
2. **Early Detection**: Logs highlight potential API or sensor issues
3. **Diagnostics**: Operators can see which metrics are failing validation
4. **Backwards Compatible**: Only affects invalid readings (normal readings unaffected)
5. **Edge Case Safety**: Handles nil pointers, out-of-range values, malformed responses

### Testing

**Validation Tests** (`pkg/collector/zone_metrics_test.go`):
- Boundary condition tests (min/max valid values)
- Out-of-range tests (too hot, too cold, etc.)
- Nil pointer handling
- Multiple validation errors
- Error message formatting

**Example**:
```go
type validationTest struct {
    name    string
    temp    float32
    wantErr bool
}

{name: "too cold", temp: -51, wantErr: true},
{name: "valid temp", temp: 20.5, wantErr: false},
{name: "too hot", temp: 61, wantErr: true},
```

---

## Observability

### Exporter Health Metrics (P2.1)

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

### Mock Strategy

- Minimal mocking (prefer real objects)
- Use nil clients for testing when real API not needed
- Custom registries for metric tests

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
- Configurable scrape timeout (default 5s)

---

## Future Enhancement Opportunities

### Could Add

1. **Circuit Breaker**: Skip API calls during outages, cache last-known metrics
2. **Retry with Exponential Backoff**: Automatic retry for transient errors
3. **Multi-Home Support**: Collect metrics from multiple homes in parallel
4. **Custom Metrics**: Allow users to add custom metric collection
5. **Webhooks**: Push alerts instead of relying on Prometheus polling

### Won't Add

- Complex caching strategies (Prometheus handles this)
- Real-time metric streaming (Prometheus is polling-based)
- Metric aggregation (use Prometheus queries instead)

---

## Deployment Recommendations

### Environment Configuration

```bash
# Minimal deployment
TADO_TOKEN_PATH=/var/lib/tado/token.json
TADO_TOKEN_PASSPHRASE=your-secure-passphrase
TADO_PORT=8080

# Optional
TADO_LOG_LEVEL=info
TADO_SCRAPE_TIMEOUT=5
TADO_HOME_ID=12345  # Filter to specific home
```

### Kubernetes Deployment

```yaml
containers:
  - name: tado-exporter
    image: tado-exporter:latest
    env:
      - name: TADO_PORT
        value: "8080"
      - name: TADO_TOKEN_PATH
        value: /secrets/tado/token.json
      - name: TADO_TOKEN_PASSPHRASE
        valueFrom:
          secretKeyRef:
            name: tado-secrets
            key: passphrase
    volumeMounts:
      - name: tado-secrets
        mountPath: /secrets/tado
        readOnly: true
```

---

## References

- [Tado API Documentation](https://support.tado.com/hc/en-us/articles/8113175915041)
- [Prometheus Client Libraries](https://prometheus.io/docs/instrumenting/clientlibs/)
- [clambin/tado GitHub](https://github.com/clambin/tado)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/instrumentation/)
