# Tado Exporter Alerts - Runbook

This document provides troubleshooting guidance for each alert in the Tado exporter monitoring system.

---

## TadoExporterDown

**Severity**: CRITICAL
**Component**: Exporter

### What It Means

The Tado exporter container/pod has stopped responding to HTTP requests. Prometheus cannot connect to the exporter's metrics endpoint.

### Why It Happens

1. Container/pod crash or OOMKill
2. Application error causing hang
3. Network connectivity lost
4. Port misconfiguration
5. Resource exhaustion

### Troubleshooting Steps

#### 1. Check Container Status

**Docker**:
```bash
docker ps | grep tado-exporter
docker logs tado-exporter | tail -50
```

**Kubernetes**:
```bash
kubectl get pods -l app=tado-exporter
kubectl logs -l app=tado-exporter --tail=50
kubectl describe pod <pod-name>
```

#### 2. Check Network Connectivity

```bash
# Test direct connection
curl -v http://<exporter-host>:9100/health

# From Prometheus pod/container
curl -v http://tado-exporter:9100/health
```

#### 3. Check Resource Usage

**Docker**:
```bash
docker stats tado-exporter
```

**Kubernetes**:
```bash
kubectl top pod <pod-name>
```

#### 4. Check Application Logs

Look for:
- Panic errors
- OOMKill messages
- Authentication failures
- Configuration errors

#### 5. Restart Container

**Docker**:
```bash
docker restart tado-exporter
```

**Kubernetes**:
```bash
kubectl delete pod <pod-name>
# Pod will be recreated automatically
```

### Prevention

- Set resource requests/limits in container
- Enable health checks with restart policy
- Monitor container logs for errors
- Use a process supervisor (Systemd, Docker restart policy, K8s)

---

## TadoExporterAuthenticationInvalid

**Severity**: CRITICAL
**Component**: Authentication

### What It Means

The Tado exporter cannot authenticate with the Tado API. This means NO metrics are being collected.

### Why It Happens

1. OAuth token expired (valid 1 year, needs refresh)
2. Wrong token passphrase provided
3. Token file corrupted or deleted
4. Tado account credentials changed
5. Tado API authentication service down

### Troubleshooting Steps

#### 1. Check Token Status

```bash
# Check if token file exists
ls -la /path/to/token.json

# Check file permissions
stat /path/to/token.json
```

#### 2. Check Token Passphrase

Verify the `TADO_TOKEN_PASSPHRASE` environment variable is set correctly:

```bash
# Should not be empty
echo $TADO_TOKEN_PASSPHRASE

# If using secrets, verify secret value
kubectl get secret tado-token -o jsonpath='{.data.passphrase}'
```

#### 3. Check Exporter Logs

```bash
docker logs tado-exporter | grep -i auth
# or
kubectl logs tado-exporter | grep -i auth
```

Look for messages like:
- `authentication failed`
- `invalid token`
- `token expired`

#### 4. Regenerate Token

If token is expired, regenerate it:

```bash
# Run exporter with --authenticate-only flag
docker run -it \
  -e TADO_TOKEN_PATH=/tokens/token.json \
  -e TADO_TOKEN_PASSPHRASE=your-passphrase \
  -v /path/to/tokens:/tokens \
  tado-exporter --authenticate-only

# For Kubernetes, port-forward and authenticate
kubectl port-forward pod/tado-exporter 9100:9100
# Then access http://localhost:9100/authenticate
```

#### 5. Verify Tado API Status

Check if Tado API is operational:
- Visit https://www.tadoapi.com/status
- Check Tado mobile app connectivity

### Prevention

- Set token expiration alerts (11 months after auth)
- Store token passphrase in secure secret management
- Backup token file
- Document token renewal procedure

---

## TadoExporterAuthenticationFailures

**Severity**: WARNING
**Component**: Authentication

### What It Means

Authentication requests to Tado API are failing, but the exporter has not yet marked authentication as completely invalid (some may still succeed).

### Why It Happens

1. Temporary API connectivity issues
2. Rate limiting by Tado API
3. Intermittent network failures
4. DNS resolution issues
5. Certificate validation failures

### Troubleshooting Steps

#### 1. Check Current Error Count

```bash
curl http://localhost:9100/metrics | grep tado_exporter_authentication_errors
```

#### 2. Check Network Connectivity

```bash
# Test DNS resolution
nslookup api.tado.com

# Test HTTPS connectivity
curl -v https://api.tado.com

# Check firewall/proxy
curl -v -x proxy.example.com:8080 https://api.tado.com
```

#### 3. Check Certificate

```bash
# Verify certificate is valid
curl -v https://api.tado.com 2>&1 | grep -i certificate
```

If certificate validation fails:
```bash
# Use alternative certificate bundle
export SSL_CERT_FILE=/path/to/ca-bundle.crt
```

#### 4. Check Rate Limiting

Look in logs for rate limit messages:
```bash
docker logs tado-exporter | grep -i "rate\|429"
```

If rate limited:
- Reduce scrape frequency in Prometheus
- Add backoff in exporter configuration
- Contact Tado support about API limits

#### 5. Check System Time

Authentication can fail if system time is wrong:

```bash
date
ntpdate -q pool.ntp.org
```

If time is wrong:
```bash
sudo ntpdate -s pool.ntp.org
# or
sudo timedatectl set-ntp true
```

### Prevention

- Monitor authentication error rate
- Set up network monitoring for API connectivity
- Enable verbose logging during troubleshooting
- Ensure time synchronization with NTP

---

## TadoExporterAuthenticationStale

**Severity**: WARNING
**Component**: Authentication

### What It Means

The exporter hasn't successfully authenticated in over 1 hour. Previous authentication may have been successful, but no new attempts have succeeded recently.

### Why It Happens

1. Exporter not running or stuck in error state
2. Authentication disabled or not being attempted
3. Tado account temporarily locked
4. Persistent network issues

### Troubleshooting Steps

#### 1. Check If Exporter Is Running

```bash
ps aux | grep tado-exporter
# or
docker ps | grep tado-exporter
```

#### 2. Force Re-authentication

```bash
# Restart exporter (will attempt fresh auth)
docker restart tado-exporter

# Check logs for auth attempts
docker logs tado-exporter | tail -20
```

#### 3. Check Exporter Configuration

Verify configuration:
```bash
docker exec tado-exporter cat /etc/tado/config.yaml
# or
kubectl exec tado-exporter -- cat /etc/tado/config.yaml
```

#### 4. Check Scrape Metrics

See if metric collection is working despite stale auth:

```bash
curl http://localhost:9100/metrics | grep tado_
```

If metrics exist but auth timestamp is old, collection may still be working.

### Prevention

- Set stricter alert for complete auth failure (5 minutes)
- Monitor auth success rate
- Implement automatic token refresh before expiry

---

## TadoExporterHighScrapeLa tency

**Severity**: WARNING
**Component**: Performance

### What It Means

Metric collection (scraping) is taking a long time. P95 latency exceeds 5 seconds.

### Why It Happens

1. Tado API response time degraded
2. Network latency high
3. Many zones/homes causing collection to take longer
4. System under high load
5. DNS resolution slow

### Troubleshooting Steps

#### 1. Check Current Latency

```bash
curl http://localhost:9100/metrics | grep tado_exporter_scrape_duration
```

Look for:
- `_bucket` (histogram buckets)
- `_sum` (total duration)
- `_count` (number of scrapes)

#### 2. Measure API Response Time

```bash
# Test Tado API directly
time curl -H "Authorization: Bearer $TOKEN" https://api.tado.com/v2/me

# Test from exporter container
docker exec tado-exporter time curl https://api.tado.com/v2/me
```

#### 3. Check System Load

```bash
top
# or
kubectl top node
```

If system is loaded:
- Reduce other workloads
- Increase resource requests
- Consider splitting into multiple exporters

#### 4. Check Number of Zones/Homes

More zones = longer collection time:

```bash
curl http://localhost:9100/metrics | grep -c "tado_temperature_measured"
```

#### 5. Enable Debug Logging

```bash
# Set log level to debug
-e TADO_LOG_LEVEL=debug

# Check logs for timing info
docker logs tado-exporter | grep duration
```

### Prevention

- Monitor latency trends
- Set scrape timeout higher than expected latency
- Optimize network path to Tado API
- Consider caching strategies

---

## TadoExporterScrapingErrors

**Severity**: WARNING
**Component**: Collection

### What It Means

Some metric collection attempts failed. Not all metrics may be available in this scrape cycle, but the exporter is still functional.

### Why It Happens

1. One zone failed but others succeeded
2. Temporary API error
3. Specific metric type unavailable
4. Zone data validation failed
5. Insufficient permissions

### Troubleshooting Steps

#### 1. Check Error Details

```bash
docker logs tado-exporter | grep -i error | tail -20
```

Look for:
- Which zone/metric failed
- Error type (validation, network, etc.)
- Timestamp when it occurred

#### 2. Check Scrape Error Counter

```bash
curl http://localhost:9100/metrics | grep tado_exporter_scrape_errors
```

Divide by scrape count to get error rate:
```
errors_total / scrape_count
```

#### 3. Verify Zone Configuration

Check which zones are configured:

```bash
curl http://localhost:9100/metrics | grep zone_id | head -10
```

#### 4. Manual Test Zone Collection

```bash
# Use Tado API directly to test zone
curl -H "Authorization: Bearer $TOKEN" \
  https://api.tado.com/v2/homes/123/zones
```

#### 5. Check Validation

If zone validation fails, check logs:

```bash
docker logs tado-exporter | grep validation
```

Look for out-of-range values or unexpected data structure.

### Prevention

- Monitor error rate trends
- Enable verbose logging for debugging
- Set up metrics validation alerts
- Document expected data ranges

---

## TadoExporterHighScrapingErrorRate

**Severity**: CRITICAL
**Component**: Collection

### What It Means

More than 10% of metric collection attempts are failing. This is a sign of a serious problem.

### Why It Happens

1. Tado API down or severely degraded
2. Authentication broken
3. Network path to API broken
4. Exporter misconfigured
5. Rate limiting by Tado API

### Troubleshooting Steps

#### 1. Immediately Check All Components

```bash
# 1. Is exporter running?
curl http://localhost:9100/health

# 2. Is Tado API reachable?
curl https://api.tado.com/v2/me -H "Authorization: Bearer $TOKEN"

# 3. Is network working?
curl https://www.google.com

# 4. Is DNS working?
nslookup api.tado.com
```

#### 2. Check Recent Error Count

```bash
curl http://localhost:9100/metrics | grep scrape_errors
```

Calculate error percentage:
```
errors_total_now - errors_total_5m_ago / scrapes_5m_ago
```

#### 3. Check Error Details

```bash
docker logs tado-exporter --since 5m
```

Look for:
- Pattern of errors
- Specific zones failing
- Type of errors (auth, network, validation)

#### 4. Check Prometheus Scrape Config

```bash
# Verify Prometheus targets
curl http://localhost:9090/api/v1/targets

# Check scrape frequency
grep -A5 "tado-exporter" /etc/prometheus/prometheus.yml
```

#### 5. Escalate if Needed

If API is unavailable:
- Check Tado API status page
- Contact Tado support
- Check platform status (cloud provider)

### Prevention

- Monitor error rate continuously
- Set alert threshold lower (5-10%)
- Implement circuit breaker to stop hammering failing API
- Have runbook readily available

---

## TadoExporterMissingMetrics

**Severity**: WARNING
**Component**: Data Quality

### What It Means

No temperature metrics are available. This likely means:
- Exporter is not collecting data
- All zones are configured but have no data
- Metrics are being filtered out by validation

### Why It Happens

1. Exporter just started (wait 1-2 minutes)
2. No zones configured in Tado account
3. All zones are filtered by configuration
4. Prometheus scrape hasn't happened yet
5. Metrics validation is too strict

### Troubleshooting Steps

#### 1. Check If Exporter Has Data

```bash
curl http://localhost:9100/metrics | grep tado_temperature
```

If nothing returned, exporter hasn't collected yet.

#### 2. Check Tado Account

Verify zones exist in Tado:

```bash
curl -H "Authorization: Bearer $TOKEN" https://api.tado.com/v2/homes/123/zones
```

#### 3. Check Exporter Logs

```bash
docker logs tado-exporter | tail -50
```

Look for:
- Successful collection messages
- Validation errors
- No zones found

#### 4. Verify Configuration

```bash
# Check if HOME_ID filter is set
echo $TADO_HOME_ID

# Check zone list
curl http://localhost:9100/metrics | grep zone_name | head
```

#### 5. Wait For First Scrape

New exporters may not have data immediately:

```bash
# Check scrape count
curl http://localhost:9100/metrics | grep scrape_duration_seconds_count
```

If count is 0, first scrape hasn't happened yet.

### Prevention

- Wait 2-3 minutes after startup before alerting
- Document expected zone configuration
- Test with at least one active zone

---

## TadoAPIUnreachable

**Severity**: CRITICAL
**Component**: API Connectivity

### What It Means

More than 50% of metric collection attempts are failing. The Tado API appears to be unreachable or severely degraded.

### Why It Happens

1. Tado API service is down
2. Network path to API is broken
3. Firewall/proxy blocking access
4. DNS not resolving
5. Certificate issues

### Troubleshooting Steps

#### 1. Check Tado API Status

- Visit https://status.tado.com
- Check Tado mobile app
- Look for service status page

#### 2. Test API Connectivity

```bash
# From exporter container
docker exec tado-exporter bash -c \
  "curl -v https://api.tado.com/v2/me -H 'Authorization: Bearer DUMMY' 2>&1"

# Check for connection refused vs. auth errors
# Connection refused = network problem
# 401/403 = network is working, auth issue
```

#### 3. Check Firewall/Proxy

```bash
# Test direct connectivity
telnet api.tado.com 443

# If behind proxy
curl -x proxy.example.com:8080 https://api.tado.com
```

#### 4. Check DNS Resolution

```bash
nslookup api.tado.com
# or
docker exec tado-exporter nslookup api.tado.com
```

If DNS fails, update resolver:
```bash
# In container, modify /etc/resolv.conf or pass --dns flag
docker run --dns 8.8.8.8 tado-exporter
```

#### 5. Check Certificate

```bash
curl -v https://api.tado.com 2>&1 | grep -i "certificate\|SSL\|TLS"

# If cert issue, may need to update CA bundle
```

### Escalation

If API is actually down:
1. Contact Tado support
2. Check public status pages
3. Wait for service recovery
4. Consider failover/caching strategy

---

## TadoExporterCircuitBreakerOpen

**Severity**: WARNING
**Component**: Resilience

### What It Means

The circuit breaker has opened due to repeated failures. The exporter is now blocking requests to Tado API to prevent cascading failures.

### Why It Happens

1. Tado API experienced extended outage
2. Authentication repeatedly failed
3. Network issues caused many failures
4. Circuit breaker threshold was exceeded

### Troubleshooting Steps

#### 1. Check Circuit Breaker State

```bash
curl http://localhost:9100/metrics | grep circuit_breaker
```

States:
- 0 = Closed (normal)
- 1 = Open (blocking requests)
- 2 = Half-Open (testing recovery)

#### 2. Fix Underlying Issue

Follow troubleshooting for the underlying error type:
- See [TadoAPIUnreachable](#tadoapiunreachable) for connectivity
- See [TadoExporterAuthenticationInvalid](#tadoexporterauthenticationinvalid) for auth

#### 3. Monitor Recovery

Circuit breaker typically opens for 1-2 minutes before half-open:

```bash
# Watch state recovery
watch "curl http://localhost:9100/metrics | grep circuit_breaker"
```

#### 4. Manual Reset (if needed)

```bash
# Restart exporter to reset circuit breaker
docker restart tado-exporter
```

### Prevention

- Monitor circuit breaker state
- Configure appropriate circuit breaker thresholds
- Have alerting for open state
- Document recovery time

---

## TadoExporterAverageScrapeDuration

**Severity**: INFO
**Component**: Performance

### What It Means

Average metric collection time has been increasing. This is just informational, not an immediate problem.

### Why It Happens

1. Tado API response times increasing
2. More zones being monitored
3. Network latency increasing
4. System load increasing

### Troubleshooting Steps

#### 1. Monitor Trend

Check if it continues to increase:

```bash
# Get historical scrape duration
curl http://localhost:9090/api/v1/query_range?query=avg(rate(tado_exporter_scrape_duration_seconds_sum[5m]))&start=...&end=...
```

#### 2. Compare to Baseline

- What was the typical duration before?
- What changed recently?
- Was more capacity added?

#### 3. Optimize If Needed

If duration is acceptable, no action needed. If trending high:
- Reduce number of zones monitored (if using filter)
- Increase resources for exporter
- Optimize network path
- Check Tado API status

### Prevention

- Establish baseline metrics during normal operation
- Monitor trends regularly
- Set thresholds based on acceptable SLA

---

## General Troubleshooting

### View All Metrics

```bash
curl http://localhost:9100/metrics
```

### Check Prometheus Scrape Config

```yaml
# /etc/prometheus/prometheus.yml
scrape_configs:
  - job_name: 'tado-exporter'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 30s
    scrape_timeout: 10s
```

### Test Alert Firing

```bash
# In Prometheus UI
/alerts

# Or query:
ALERTS{alertname="TadoExporterDown"}
```

### Enable Debug Logging

```bash
docker run \
  -e TADO_LOG_LEVEL=debug \
  tado-exporter
```

### Common Commands

```bash
# Health check
curl http://localhost:9100/health

# Get all metrics
curl http://localhost:9100/metrics

# Get specific metric
curl http://localhost:9100/metrics | grep "metric_name"

# Count metrics
curl http://localhost:9100/metrics | grep -c "^tado_"
```

---

## Support & Escalation

### When to Escalate

1. **Network issues**: Contact network/infrastructure team
2. **Tado API down**: Contact Tado support
3. **Exporter bugs**: File GitHub issue
4. **Prometheus issues**: Check Prometheus logs

### Getting Help

- Check exporter logs: `docker logs tado-exporter`
- Check Prometheus logs
- Review this runbook
- GitHub issues: https://github.com/andreweacott/tado-prometheus-exporter/issues
