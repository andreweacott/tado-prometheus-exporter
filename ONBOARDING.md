# Tado Prometheus Exporter - Developer Onboarding Guide

**Analysis Date:** November 4, 2025

> This guide helps developers understand the codebase quickly and become productive. It focuses on the "why" and "how" rather than just the "what" that's already in the code.

---

## Overview

**What is this?** A Prometheus exporter that collects metrics from Tado smart heating systems.

**Key Innovation:** Uses OAuth 2.0 device code grant flow (the "visit this URL" authentication) with encrypted token storage, enabling unattended operation without requiring upfront OAuth app registration. This is different from other tado prometheus exporters that use an outdated (and now unsupported) username/password based authentication.

**Architecture Philosophy:** Simple, fault-tolerant, observable. The system continues collecting partial metrics even when some API calls fail—crucial for production monitoring.

**Key Architectural Patterns:**
1. **Circuit Breaker** - Protects against API failures with automatic recovery
2. **Graceful Degradation** - Collects partial metrics when some API calls fail
3. **Metric Validation** - Validates data quality before recording to Prometheus
4. **Zero-Config Auth** - OAuth device code flow with encrypted token storage
5. **Self-Monitoring** - Exposes exporter health metrics for observability

---

## Getting Started

### Prerequisites

- **Go 1.23+** (note: go.mod specifies 1.25.1, but 1.23+ should work)
- **Docker** (optional, for containerized deployment)
- **A Tado account**

### Quick Start (5 minutes)

```bash
# 1. Clone and build
git clone git@github.com:andreweacott/tado-prometheus-exporter.git
cd tado-prometheus-exporter
make build

# 2. Run (you'll be prompted to authenticate on first run)
make run TOKEN_PASSPHRASE=test-secret-123

# 3. Verify it's working
curl http://localhost:9100/health
curl http://localhost:9100/metrics
```

**First Run Experience:** When you run it for the first time, you'll see:
```
No token found. Visit this link to authenticate:
https://my.tado.com/oauth/authorize?code=XXXX&device_code=YYYY
```

Visit that URL, authenticate, and the exporter will automatically save the encrypted token. Subsequent runs won't require re-authentication.

### Development Workflow

```bash
# Run tests (fast feedback loop)
make test

# Run with coverage
make test-coverage
make coverage  # Opens HTML report

# Full check before committing
make check     # Runs: build + lint + vet + test

# Clean build artifacts
make clean
```

**Hot Tip:** Use `make run` during development—it rebuilds automatically before running.

---

## Architecture & Code Organization

### Directory Structure

```
├── cmd/exporter/          # Main application entry point
│   ├── main.go           # Startup, initialization, orchestration
│   └── server.go         # HTTP server, endpoints, graceful shutdown
│
├── pkg/
│   ├── auth/             # OAuth2 authentication & token management
│   ├── collector/        # Prometheus collector (fetches Tado metrics)
│   ├── config/           # Configuration (flags, env vars, validation)
│   ├── logger/           # Structured logging wrapper
│   └── metrics/          # Prometheus metric definitions
│
├── .github/workflows/    # CI/CD (test.yaml, build.yaml)
├── Dockerfile            # Multi-stage build for small images
├── local/                # LOCAL TESTING ONLY: Docker Compose stack and Prometheus config
│   ├── docker-compose.yml    # Complete observability stack (exporter + Prometheus + Grafana)
│   └── prometheus.yml        # Prometheus scrape configuration for local testing
└── Makefile             # Development commands (build, test, lint, run)
```

### Package Responsibilities

| Package | What It Does | Key Files | Why It Matters |
|---------|--------------|-----------|----------------|
| `cmd/exporter` | Application entry point, server lifecycle | `main.go`, `server.go` | Where execution starts; orchestrates everything |
| `pkg/auth` | OAuth2 device code flow, token encryption | `authenticator.go` | The "magic" that enables zero-config auth |
| `pkg/collector` | Implements Prometheus collector interface | `collector.go`, `circuit_breaker.go`, `zone_metrics.go` | Heart of metrics collection; implements graceful degradation + resilience |
| `pkg/config` | CLI flags + env vars with precedence | `config.go` | Precedence: CLI > env > defaults |
| `pkg/logger` | Structured logging (logrus wrapper) | `logger.go` | Adds context fields like home_id, zone_id |
| `pkg/metrics` | Metric definitions (Tado + exporter health) | `metrics.go`, `exporter_metrics.go` | Defines what we expose to Prometheus |

### Data Flow

```
[Prometheus Scrape] → /metrics endpoint
                    ↓
              [HTTP Server]
                    ↓
            [TadoCollector.Collect()]
                    ↓
        [Fetch from Tado API via OAuth client]
                    ↓
        [Update Prometheus metric values]
                    ↓
        [Return metrics to Prometheus]
```

**Critical Detail:** Metrics are fetched **on-demand** when Prometheus scrapes `/metrics`. There's no background polling. This is intentional—simpler, more predictable, and aligns with Prometheus's pull model.

---

## Key Concepts & Patterns

### 1. OAuth2 Device Code Flow

**Why it's important:** Most OAuth apps require you to register an app, get client credentials, and configure redirect URIs. This uses the **device code grant**, which only requires:
1. A passphrase (to encrypt the token)
2. User visits a URL and authorizes once
3. Refresh token is stored and used to continue accessing the API

**Where it happens:** `pkg/auth/authenticator.go` → delegates to `github.com/clambin/tado/v2` library

**The library does:**
- Initiates device code flow
- Polls for user authorization
- Stores encrypted token with your passphrase
- Automatically refreshes token when expired

**Your job as developer:** Just pass `tokenPath` and `tokenPassphrase`. The library handles everything else.

### 2. Graceful Degradation (Partial Metrics Collection)

**The Problem:** If one Tado API call fails (e.g., fetching weather), should the entire scrape fail?

**The Solution:** Continue collecting what you can. The collector catches errors, logs them, and moves on.

**See it in action:** `pkg/collector/collector.go` → `fetchAndCollectMetrics()`

```go
// Pattern: Continue on error, track failures
if err := tc.collectHomeMetrics(ctx, homeID); err != nil {
    log.Warn("Failed to collect home metrics", "error", err)
    collectionErrors = append(collectionErrors, err)
    // CONTINUE to zone metrics instead of returning
}
```

**Why it matters:** Operators get partial metrics even during API issues → better debugging, alerting still works.

### 3. Configuration Precedence (CLI > Env > Defaults)

**The Rule:** CLI flags override environment variables, which override defaults.

**Example:**
```bash
# Env var sets port to 8080
export TADO_PORT=8080

# CLI flag overrides to 9100
./tado-exporter --port=9100  # Uses 9100, not 8080
```

**Where it's implemented:** `pkg/config/config.go` → `Load()` function

**Why this pattern:** Supports multiple deployment scenarios:
- Docker: Environment variables in local/docker-compose.yml
- Kubernetes: ConfigMaps + CLI args
- Local dev: CLI flags for quick iteration

### 4. Exporter Health Metrics (Self-Monitoring)

**The exporter monitors itself** and exposes these metrics:

- `tado_exporter_scrape_duration_seconds` - How long scrapes take
- `tado_exporter_scrape_errors_total` - Counter of failed scrapes
- `tado_exporter_authentication_valid` - Is auth token valid? (1=yes, 0=no)
- `tado_exporter_authentication_errors_total` - Auth failure count
- `tado_exporter_last_authentication_success_unix` - Timestamp of last successful auth

**Why:** You can alert on `tado_exporter_authentication_valid == 0` to detect auth issues before metrics stop flowing.

### 5. Circuit Breaker Pattern (Resilience Against API Failures)

**The Problem:** If the Tado API becomes unreliable, should we keep hammering it with requests?

**The Solution:** Use a circuit breaker that "trips" after repeated failures, temporarily stops making requests, then gradually tries again.

**Where it happens:** `pkg/collector/circuit_breaker.go` wraps the Tado API client

**How it works:**
- **Closed** (normal): All requests go through
- **Open** (tripped): After 5 consecutive failures, circuit opens for 30 seconds
- **Half-Open** (testing): After timeout, allows 1 request to test if API recovered

```go
// In main.go initialization
cbConfig := collector.DefaultCircuitBreakerConfig()
tadoClient = collector.NewTadoAPIWithCircuitBreaker(tadoClient, cbConfig)
```

**Configuration:**
- `MaxConsecutiveFailures`: 5 (configurable)
- `Timeout`: 30 seconds (how long to wait before testing recovery)

**Why it matters:** Prevents cascading failures, gives the API time to recover, provides clear error messages to operators.

### 6. Metric Validation (Data Quality Assurance)

**The Problem:** What if the Tado API returns nonsensical values (e.g., temperature = 500°C)?

**The Solution:** Validate all metrics against known valid ranges before recording to Prometheus.

**Where it happens:** `pkg/collector/zone_metrics.go`

**Validation Ranges:**
- Temperature: -50°C to 60°C (typical building range)
- Humidity: 0% to 100% (by definition)
- Heating Power: 0% to 100% (percentage range)

**Behavior on invalid data:**
- Log a warning with context (zone_id, metric name, value, reason)
- Skip recording that specific metric
- Continue collecting other metrics (graceful degradation)

**Why it matters:** Prevents bad data from corrupting Prometheus time-series, helps operators spot API or sensor issues early.

### 7. Test Isolation with Custom Registries

**The Problem:** Prometheus metrics are global singletons. Creating the same metric twice in tests causes registration conflicts.

**The Solution (in production):** Metrics are registered once at startup.

**The Solution (in tests):** Some tests are skipped to avoid re-registration issues. Tests that need isolation use custom registries:

```go
registry := prometheus.NewRegistry()  // Local registry for test
registry.Register(metric)
```

**See:** `pkg/collector/collector_test.go` - several tests are explicitly skipped with explanations.

---

## How To: Common Tasks

### Add a New Tado Metric

**Scenario:** You want to expose a new metric from the Tado API.

**Steps:**

1. **Define the metric** in `pkg/metrics/metrics.go`:
   ```go
   NewMetricName: *prometheus.NewGaugeVec(
       prometheus.GaugeOpts{
           Name: "tado_new_metric_name",
           Help: "Description of what this measures",
       },
       []string{"home_id", "zone_id", "zone_name", "zone_type"},
   ),
   ```

2. **Register it** in the `Register()` method (same file):
   ```go
   if err := prometheus.DefaultRegisterer.Register(&md.NewMetricName); err != nil {
       return err
   }
   ```

3. **Collect the data** in `pkg/collector/collector.go`:
   - Add fetch logic in `collectHomeMetrics()` or `collectZoneMetrics()`
   - Update metric: `tc.metricDescriptors.NewMetricName.WithLabelValues(...).Set(value)`

4. **Add to Describe/Collect** methods in `collector.go`:
   ```go
   // In Describe()
   tc.metricDescriptors.NewMetricName.Describe(ch)

   // In Collect()
   tc.metricDescriptors.NewMetricName.Collect(ch)
   ```

5. **Write a test** in `pkg/collector/collector_test.go`

6. **Document** in README.md

**Gotcha:** If you get Prometheus registration errors in tests, consider using a custom registry or skipping the test.

### Add Metric Validation

**Scenario:** You're adding a new metric and want to ensure data quality.

**Steps:**

1. **Define validation constants** in `pkg/collector/zone_metrics.go`:
   ```go
   const (
       MinValidMyMetric float32 = 0
       MaxValidMyMetric float32 = 100
   )
   ```

2. **Create validation function**:
   ```go
   func validateMyMetric(value float32, fieldName string) error {
       if value < MinValidMyMetric || value > MaxValidMyMetric {
           return &ValidationError{
               Field: fieldName,
               Value: value,
               Reason: fmt.Sprintf("outside valid range [%g, %g]", MinValidMyMetric, MaxValidMyMetric),
           }
       }
       return nil
   }
   ```

3. **Add to ZoneMetrics struct**:
   ```go
   type ZoneMetrics struct {
       // ... existing fields
       MyMetricValue *float32
   }
   ```

4. **Extract in ExtractAllZoneMetrics**:
   ```go
   func ExtractAllZoneMetrics(zoneState *tado.ZoneState) *ZoneMetrics {
       return &ZoneMetrics{
           // ... existing fields
           MyMetricValue: extractMyMetric(zoneState),
       }
   }
   ```

5. **Validate before recording** in `pkg/collector/collector.go`:
   ```go
   if metrics.MyMetricValue != nil {
       if err := validateMyMetric(*metrics.MyMetricValue, "my_metric"); err != nil {
           tc.log.Warn("Invalid metric, skipping", "error", err.Error())
       } else {
           tc.metricDescriptors.MyMetric.WithLabelValues(...).Set(float64(*metrics.MyMetricValue))
       }
   }
   ```

6. **Write validation tests** in `pkg/collector/zone_metrics_test.go`

**Why validate:** Prevents bad data from corrupting Prometheus time-series, enables early detection of API/sensor issues.

### Configure Circuit Breaker

**Scenario:** You need to adjust circuit breaker behavior for your environment.

**Default Configuration:**
- Max failures before opening: 5
- Open timeout: 30 seconds
- Max requests in half-open: 1

**To Change:**

Edit `cmd/exporter/main.go`:

```go
cbConfig := collector.CircuitBreakerConfig{
    MaxConsecutiveFailures: 3,  // Open after 3 failures instead of 5
    Timeout:                60 * time.Second,  // Wait 60s before testing recovery
}
tadoClient = collector.NewTadoAPIWithCircuitBreaker(tadoClient, cbConfig)
```

**When to adjust:**
- **Lower MaxFailures** (2-3): If API is critical and you want fast failure detection
- **Higher MaxFailures** (10+): If API has occasional transient errors
- **Longer Timeout** (60s+): If API takes time to recover
- **Shorter Timeout** (10s): If API recovers quickly

**Monitoring:** Watch logs for "circuit breaker is open" messages to tune these values.

### Run Tests

```bash
# All tests
make test

# Specific package
go test -v ./pkg/config/

# With race detector (slower, catches concurrency bugs)
go test -v -race ./...

# With coverage
make test-coverage
go tool cover -html=coverage.out
```

**Current Coverage:** ~80+ tests across all packages

**Testing Philosophy:** Unit tests for logic, minimal mocking (prefer real objects with nil clients), custom registries for metric tests.

### Debug Authentication Issues

**Problem:** Token not working, can't authenticate, etc.

**Steps:**

1. **Enable debug logging:**
   ```bash
   ./exporter --log-level=debug --token-passphrase=your-secret
   ```

2. **Check token file:**
   ```bash
   ls -la ~/.tado-exporter/token.json
   # Should exist and have restrictive permissions
   ```

3. **Manually delete token to re-authenticate:**
   ```bash
   rm ~/.tado-exporter/token.json
   ./exporter --token-passphrase=your-secret
   # Will prompt for re-authentication
   ```

4. **Check authentication metrics:**
   ```bash
   curl http://localhost:9100/metrics | grep tado_exporter_authentication
   ```

**Common Issues:**
- Wrong passphrase → can't decrypt token → re-authentication triggered
- Token file permissions too open → security error
- Network issues → device code flow times out (5 min limit)

### Build and Deploy with Docker

```bash
# Build image
make docker-build

# Run container (production-ready)
make docker-run TOKEN_PASSPHRASE=your-secret
```

**Dockerfile Details:**
- Multi-stage build (golang:1.25 builder → alpine:latest runtime)
- Binary is statically linked (CGO_ENABLED=0)
- Includes CA certificates for HTTPS
- Health check on /health endpoint
- Final image is tiny (~10-20 MB)

### Local Testing with Docker Compose

For local development and testing with a complete observability stack (exporter + Prometheus + Grafana):

```bash
# Start the full local stack (LOCAL TESTING ONLY - not for production)
cd local && TADO_TOKEN_PASSPHRASE=your-secret docker-compose up -d

# View logs
cd local && docker-compose logs -f exporter

# Access services
# - Exporter metrics: http://localhost:9100/metrics
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
```

**⚠️ Important:** `local/docker-compose.yml` is for local development and testing only. For production deployment:
- Use standalone `docker run` or orchestration platforms (Kubernetes, Docker Swarm)
- Set up your own Prometheus and Grafana instances separately
- Use proper secrets management and resource constraints
- Configure persistent storage appropriately for your environment

### Add a Configuration Option

**Steps:**

1. **Add field to Config struct** in `pkg/config/config.go`:
   ```go
   type Config struct {
       NewOption string
   }
   ```

2. **Add environment variable and flag** in `Load()` method:
   ```go
   envNewOption := os.Getenv("TADO_NEW_OPTION")
   // ...
   fs.StringVar(&cfg.NewOption, "new-option", envNewOption, "Description")
   ```

3. **Add validation** in `Validate()` method if needed

4. **Write tests** in `pkg/config/config_test.go`

5. **Document** in README.md

---

## Dependencies & External Services

### Key Dependencies

| Dependency | Version | Purpose | Why This One? |
|------------|---------|---------|---------------|
| `clambin/tado/v2` | v2.6.2 | Tado API client | OAuth2 device code flow + auto token refresh + encrypted storage |
| `prometheus/client_golang` | v1.23.2 | Prometheus client library | Official Prometheus library |
| `sirupsen/logrus` | v1.9.3 | Structured logging | Industry standard, simple structured logging |
| `sony/gobreaker` | v1.0.0 | Circuit breaker | Resilience pattern for API failures |
| `stretchr/testify` | v1.11.1 | Test assertions | Makes tests readable with assert/require |
| `golang.org/x/oauth2` | v0.32.0 | OAuth2 client | Required by clambin/tado |

### Tado API

**Authentication:** OAuth2 device code grant
- No client ID/secret required
- User authorizes via browser
- Token encrypted and stored locally

**Endpoints Used:**
- `GET /api/v2/me` - Get user info and homes
- `GET /api/v2/homes/{homeId}/state` - Home state (presence)
- `GET /api/v2/homes/{homeId}/weather` - Weather data
- `GET /api/v2/homes/{homeId}/zones` - List zones
- `GET /api/v2/homes/{homeId}/zoneStates` - Zone states (temperature, humidity, etc.)

**Rate Limits:** Not explicitly documented by Tado. Default scrape interval is 5 minutes (Prometheus recommendation).

**Library Abstraction:** `clambin/tado/v2` wraps all API calls. You don't make raw HTTP requests—use the library's typed methods.

---

## Development Conventions

### Code Style

- **Standard Go conventions** (gofmt, go vet)
- **Linting:** golangci-lint (run with `make lint`)
- **Imports:** Group in three sections: stdlib, external, internal
- **Error handling:** Always wrap errors with context using `fmt.Errorf("context: %w", err)`

### Logging

**Use structured logging** with context fields:

```go
log.Info("Collecting metrics", "home_id", homeID, "zone_count", zoneCount)
log.Warn("Failed to fetch weather", "home_id", homeID, "error", err.Error())
log.Error("Authentication failed", "error", err.Error())
```

**Log Levels:**
- `debug` - Detailed flow, API responses (verbose)
- `info` - Normal operations (startup, config, successful collections)
- `warn` - Recoverable errors (partial collection failures)
- `error` - Serious errors (auth failures, startup failures)

**Never log secrets:** Passphrases, tokens, API credentials

### Testing

**Pattern: Table-driven tests**

```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"test case 1", "input1", "output1"},
    {"test case 2", "input2", "output2"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := myFunction(tt.input)
        assert.Equal(t, tt.expected, result)
    })
}
```

**Use testify assertions:**
- `require.NoError(t, err)` - Fail immediately if error
- `assert.Equal(t, expected, actual)` - Continue on failure
- `assert.NotNil(t, obj)` - Check for nil

**Test files naming:** `*_test.go` in same package

### Commit Messages

- Use imperative mood: "Add feature" not "Added feature"
- First line summary (50 chars max)
- Blank line
- Detailed explanation if needed

---

## Key Insights & Gotchas

### Design Decisions

**1. On-Demand Metrics (Not Background Polling)**

**Why:** Aligns with Prometheus pull model. Simpler than background goroutines + caching. Less memory usage.

**Trade-off:** Each scrape hits Tado API (but configurable via Prometheus scrape_interval).

**2. Partial Collection on Failure**

**Why:** Better operator experience. Partial metrics > no metrics.

**Implementation:** Catch errors, log them, continue to next metric.

**3. No Metric Caching**

**Why:** Prometheus handles caching and staleness detection. Adding caching here would be redundant and error-prone.

**4. Single Goroutine for Collection**

**Why:** Simplicity. Tado API is fast enough. No need for concurrent API calls yet.

**When to reconsider:** If you have many homes (10+) and scrapes are too slow.

**5. Circuit Breaker for API Resilience**

**Why:** Protects against cascading failures when Tado API becomes unreliable. Gives the API time to recover instead of hammering it with requests.

**Trade-off:** Some scrapes will fail completely when circuit is open, but this prevents worse outcomes (prolonged outages, rate limiting, resource exhaustion).

**Implementation:** Wraps all API calls with `sony/gobreaker` library. Transparent to collector logic.

**6. Metric Validation at Collection Time**

**Why:** Better to catch bad data early than to corrupt Prometheus time-series. Also helps operators spot sensor/API issues.

**Trade-off:** Slight performance overhead for validation checks, but negligible compared to API latency.

**Implementation:** Extract → Validate → Record. Invalid metrics are skipped with warnings logged.

### Non-Obvious Behavior

**1. Token Path Default Depends on HOME**

The default token path is `~/.tado-exporter/token.json`, but in Docker, `HOME=/root`, so it becomes `/root/.tado-exporter/token.json`.

**Why it matters:** Volume mounts need to match. In the local docker-compose stack, the exporter runs as root, so tokens go to `/root/.tado-exporter`. For production (non-root user), tokens go to `/home/exporter/.tado-exporter`. See local/docker-compose.yml for the local example.

**2. Prometheus Registration is Global**

Metrics are registered with `prometheus.DefaultRegisterer` at startup. You can't register the same metric twice.

**Why it matters:** Tests that create multiple `MetricDescriptors` will fail. Use custom registries in tests or skip.

**3. Graceful Shutdown Waits for Current Scrape**

When you SIGTERM the exporter, it waits up to 10 seconds for the current HTTP request (metrics scrape) to complete.

**Why it matters:** In Kubernetes, set `terminationGracePeriodSeconds` >= 15 to account for scrape timeout + shutdown timeout.

**4. Zone-Level Metrics Have 4 Labels**

All zone metrics have: `home_id`, `zone_id`, `zone_name`, `zone_type`

**Why it matters:** High cardinality if you have many zones. Prometheus can handle hundreds of zones, but be aware.

**5. Authentication Happens Synchronously at Startup**

If no valid token exists, the exporter blocks on authentication (waiting for user to visit URL).

**Why it matters:** First startup in automation (CI, Kubernetes) will hang until you authenticate. Pre-authenticate before deploying.

**6. Circuit Breaker State is Not Persisted**

The circuit breaker resets to "closed" on exporter restart, even if it was open before.

**Why it matters:** Restarting the exporter will bypass the circuit breaker protection temporarily. Don't restart the exporter as a "fix" for circuit breaker issues—you'll just hit the same failures again.

**Better approach:** Wait for the circuit to self-heal, or fix the underlying API issue.

### Modern vs Legacy Patterns

**Current (Modern) Patterns:**
- OAuth2 device code flow (no pre-registered app)
- Circuit breaker for API resilience
- Metric validation before recording
- Graceful degradation on errors
- Structured logging with context
- Self-monitoring (exporter health metrics)
- Multi-stage Docker builds
- Table-driven tests
- Separation of concerns (extraction vs validation in zone_metrics.go)

**What to Avoid:**
- Hard-coded credentials (always use env vars or CLI flags)
- Panicking on errors (use proper error handling)
- Global variables (except for metrics, which are Prometheus convention)
- Unmarshaled JSON maps (use the library's typed structs)
- Recording unvalidated metrics to Prometheus

---

## CI/CD & Release Process

### GitHub Actions Workflows

**`.github/workflows/test.yaml`** - Runs on every push/PR
- Checks out code
- Sets up Go
- Runs `go test ./...`
- Uploads coverage

**`.github/workflows/build.yaml`** - Runs on tags/releases
- Builds Docker image
- Pushes to GitHub Container Registry (ghcr.io)

### Release Process

1. **Update version** (if applicable)
2. **Run full check:** `make check`
3. **Tag release:** `git tag v1.0.0 && git push --tags`
4. **GitHub Actions** builds and pushes Docker image
5. **Users pull:** `docker pull ghcr.io/andreweacott/tado-prometheus-exporter:latest`

**See:** `RELEASE.md` for detailed release process (if present)

---

## Troubleshooting

### Common Issues

**1. "Failed to create OAuth2 client"**

**Causes:**
- Wrong passphrase for existing token
- Token file corrupted
- Network issues

**Fix:**
```bash
rm ~/.tado-exporter/token.json
./exporter --token-passphrase=your-secret
# Re-authenticate
```

**2. "Prometheus registration error"**

**Cause:** Trying to register the same metric twice (usually in tests)

**Fix:** Use custom registries in tests or skip the test

**3. "No metrics returned"**

**Causes:**
- Home ID filter doesn't match your homes
- Authentication invalid
- Tado API down

**Debug:**
```bash
# Check auth status
curl http://localhost:9100/metrics | grep authentication_valid

# Check scrape errors
curl http://localhost:9100/metrics | grep scrape_errors_total

# Enable debug logs
./exporter --log-level=debug --token-passphrase=secret
```

**4. "Context deadline exceeded"**

**Cause:** Scrape timeout too short for your network/API response time

**Fix:**
```bash
./exporter --scrape-timeout=20  # Increase from default 10s
```

**5. "Circuit breaker is open"**

**Cause:** Tado API failed multiple times (5 consecutive failures by default)

**What it means:** The exporter is temporarily stopping API requests to give the API time to recover

**Fix:**
- **Wait:** Circuit will automatically test recovery after 30 seconds
- **Check Tado API status:** Visit Tado's status page or Twitter
- **Check logs:** Look for underlying error messages before circuit opened
- **Adjust threshold:** If transient errors are common, increase `MaxConsecutiveFailures`

**Debug:**
```bash
# Check for circuit breaker errors
./exporter --log-level=debug --token-passphrase=secret 2>&1 | grep "circuit breaker"

# Common patterns in logs:
# "circuit breaker is open" → Circuit tripped, waiting for timeout
# "circuit breaker is half-open" → Testing if API recovered
# "circuit breaker is closed" → Normal operation resumed
```

### Useful Commands

```bash
# Check metrics endpoint
curl http://localhost:9100/metrics

# Check health
curl http://localhost:9100/health

# View token file (encrypted, won't be readable)
cat ~/.tado-exporter/token.json

# Run with debug logging
./exporter --log-level=debug --token-passphrase=secret

# Test authentication without running server
# (Not currently supported, but you could add a --auth-only flag)
```

---

## Open Questions & Uncertainties

### Areas Needing Clarification

**1. Tado API Rate Limits**

- **Unknown:** Official rate limits from Tado
- **Current approach:** Default 5-minute Prometheus scrape interval
- **Risk:** Too frequent scraping might trigger throttling
- **Next step:** Monitor in production, reach out to Tado support for guidance

**2. Multi-Home Performance**

- **Unknown:** How well does this scale with many homes (10+)?
- **Current:** Single-threaded collection (one home at a time)
- **Consideration:** Could parallelize home collection if needed
- **Next step:** Test with multi-home accounts

**3. Token Refresh Timing**

- **Assumption:** `clambin/tado/v2` handles automatic token refresh
- **Unknown:** Exact timing, refresh logic, failure behavior
- **Current:** Trust the library
- **Next step:** Read library source or test with long-running deployments

**4. Metric Cardinality**

- **Unknown:** What's the practical limit of zones per home?
- **Current:** Each zone creates 8 labeled metrics (4 labels each)
- **Calculation:** 100 zones = 800 metrics (probably fine)
- **Risk:** 1000+ zones might strain Prometheus
- **Next step:** Document recommended limits based on real usage

**5. Kubernetes Deployment Best Practices**

- **Unknown:** Detailed Kubernetes manifests (Deployment, Service, ConfigMap, Secret)
- **Next step:** Create `k8s/` directory with example manifests

**6. Circuit Breaker Metrics**

- **Gap:** Circuit breaker state not currently exposed as Prometheus metrics
- **Opportunity:** Add `tado_exporter_circuit_breaker_state` gauge (0=closed, 1=open, 2=half-open)
- **Next step:** Enhance circuit_breaker.go to expose state metrics

### Questions for New Contributors

- Are there other Tado API endpoints we should expose?
- Should we add a `/config` endpoint to view current configuration?
- Should we support multiple Tado accounts in one exporter instance?
- Should circuit breaker state be exposed as a Prometheus metric?

---

## Additional Resources

### Official Documentation

- **README.md** - Quick start, usage, metrics reference

### Pre-Built Monitoring Resources

**📊 Grafana Dashboard** (`docs/examples/dashboards/tado-exporter.json`)
- Comprehensive monitoring: authentication status, temperatures, humidity, heating power
- 24-hour trends with mean/max/min aggregations
- Exporter health panels (scrape duration, errors, latency)
- Weather data (outside temp, solar intensity, presence)
- Import via Grafana UI or API - see `docs/examples/dashboards/README.md`

### External Links

- [Tado API Documentation](https://support.tado.com/hc/en-us/articles/8113175915041)
- [Prometheus Client Library](https://prometheus.io/docs/instrumenting/clientlibs/)
- [OAuth2 Device Code Flow](https://oauth.net/2/device-flow/)
- [clambin/tado GitHub](https://github.com/clambin/tado)
- [sony/gobreaker GitHub](https://github.com/sony/gobreaker)

### Getting Help

- **Issues:** Check existing issues in the repository
- **Logs:** Always run with `--log-level=debug` when debugging
- **Metrics:** Check `tado_exporter_*` metrics for health status

---

## Quick Reference

### Environment Variables

```bash
TADO_TOKEN_PASSPHRASE    # Required: encryption passphrase
TADO_TOKEN_PATH          # Optional: token storage path
TADO_PORT                # Optional: HTTP server port (default: 9100)
TADO_HOME_ID             # Optional: filter to specific home
TADO_SCRAPE_TIMEOUT      # Optional: API timeout in seconds (default: 10)
TADO_LOG_LEVEL           # Optional: debug|info|warn|error (default: info)
```

### Makefile Targets

```bash
make help          # Show all available targets
make build         # Build the binary
make test          # Run tests
make test-coverage # Run tests with coverage
make coverage      # Generate HTML coverage report
make lint          # Run linter
make check         # Full check (build + lint + test)
make run           # Build and run (requires TOKEN_PASSPHRASE)
make docker-build  # Build Docker image
make docker-run    # Run Docker container
make clean         # Remove build artifacts
```

### HTTP Endpoints

- `GET /metrics` - Prometheus metrics
- `GET /health` - Health check (returns JSON: `{"status":"ok"}`)

### Key Files to Know

**Core Application:**
- `cmd/exporter/main.go` - Application entry point, initialization sequence
- `cmd/exporter/server.go` - HTTP server, graceful shutdown

**Authentication & API:**
- `pkg/auth/authenticator.go` - OAuth2 device code flow

**Metrics Collection:**
- `pkg/collector/collector.go` - Main Prometheus collector, orchestration
- `pkg/collector/circuit_breaker.go` - API resilience wrapper
- `pkg/collector/zone_metrics.go` - Metric extraction and validation
- `pkg/collector/interfaces.go` - TadoAPI interface definition

**Configuration & Observability:**
- `pkg/config/config.go` - Configuration management
- `pkg/logger/logger.go` - Structured logging
- `pkg/metrics/metrics.go` - Tado metric definitions
- `pkg/metrics/exporter_metrics.go` - Exporter health metrics

**Operations:**
- `Makefile` - Development commands
- `Dockerfile` - Container image for deployment
- `local/docker-compose.yml` - Local testing stack (NOT for production)
- `local/prometheus.yml` - Prometheus scrape configuration for local testing
- `docs/examples/dashboards/tado-exporter.json` - Grafana dashboard

---

**Welcome to the project! 🚀**

If you have questions or find gaps in this guide, please open an issue or submit a PR to improve it.
