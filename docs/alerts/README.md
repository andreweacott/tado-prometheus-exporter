# Prometheus Alerting Rules - Tado Exporter

This directory contains pre-configured Prometheus alert rules for monitoring the Tado exporter.

## Files Included

- **`tado-exporter.yml`** - Alert rule definitions for Prometheus
- **`RUNBOOK.md`** - Troubleshooting guide for each alert

## Quick Start

### Option 1: Prometheus Configuration File

1. Copy `tado-exporter.yml` to your Prometheus alert rules directory:
   ```bash
   cp tado-exporter.yml /etc/prometheus/rules/
   ```

2. Update `/etc/prometheus/prometheus.yml`:
   ```yaml
   rule_files:
     - "rules/tado-exporter.yml"

   alerting:
     alertmanagers:
       - static_configs:
           - targets:
               - localhost:9093  # Alertmanager address
   ```

3. Reload Prometheus:
   ```bash
   # Via HTTP API
   curl -X POST http://localhost:9090/-/reload

   # Or restart
   systemctl restart prometheus
   ```

### Option 2: Docker Compose

```yaml
version: '3'
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts/tado-exporter.yml:/etc/prometheus/rules/tado-exporter.yml
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
```

### Option 3: Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-rules
  namespace: monitoring
data:
  tado-exporter.yml: |
    <contents of tado-exporter.yml>

---
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: tado-exporter
  namespace: monitoring
spec:
  groups:
    - name: tado-exporter
      interval: 30s
      rules:
        - <rules from tado-exporter.yml>
```

## Alert Rules Overview

### Critical Alerts (Immediate Action Required)

| Alert | Condition | Action |
|-------|-----------|--------|
| `TadoExporterDown` | Exporter unreachable for 2m | Restart exporter/check infrastructure |
| `TadoExporterAuthenticationInvalid` | Auth invalid for 5m | Regenerate token or check credentials |
| `TadoExporterHighScrapingErrorRate` | >10% errors for 10m | Check API connectivity or auth |
| `TadoAPIUnreachable` | >50% failures for 5m | Check API status and network connectivity |

### Warning Alerts (Monitor and Investigate)

| Alert | Condition | Action |
|-------|-----------|--------|
| `TadoExporterAuthenticationFailures` | Auth errors in 5m | Monitor, may be transient |
| `TadoExporterAuthenticationStale` | No auth for 1+ hours | Check if exporter is running |
| `TadoExporterHighScrapeLa tency` | P95 latency >5s for 10m | Monitor API performance |
| `TadoExporterScrapingErrors` | Collection failures in 5m | Check logs for details |
| `TadoExporterMissingMetrics` | No temp metrics for 10m | Wait for first scrape or check config |
| `TadoExporterCircuitBreakerOpen` | Circuit open for 1m | Fix underlying issue, wait for recovery |

### Info Alerts (Tracking Trends)

| Alert | Condition |
|-------|-----------|
| `TadoExporterAverageScrapeDuration` | Avg collection >2s for 5m |

## Configuring Alertmanager

### Basic Alertmanager Configuration

Create `/etc/alertmanager/alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m
  slack_api_url: 'YOUR_SLACK_WEBHOOK_URL'

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  routes:
    - match:
        severity: critical
      receiver: 'critical'
      continue: true
    - match:
        severity: warning
      receiver: 'warnings'

receivers:
  - name: 'default'
    slack_configs:
      - channel: '#monitoring'
        title: 'Alert: {{ .GroupLabels.alertname }}'

  - name: 'critical'
    slack_configs:
      - channel: '#critical-alerts'
        title: 'CRITICAL: {{ .GroupLabels.alertname }}'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'

  - name: 'warnings'
    slack_configs:
      - channel: '#warnings'
        title: 'Warning: {{ .GroupLabels.alertname }}'
```

### Email Notifications

```yaml
receivers:
  - name: 'email'
    email_configs:
      - to: 'ops-team@example.com'
        from: 'prometheus@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'prometheus'
        auth_password: 'password'
        headers:
          Subject: '{{ .GroupLabels.alertname }}'
```

### PagerDuty Integration

```yaml
receivers:
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'YOUR_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}: {{ .Alerts.Firing | len }} firing'
```

## Testing Alerts

### Manual Alert Testing

1. **Trigger TadoExporterDown**:
   ```bash
   # Stop the exporter
   docker stop tado-exporter

   # Wait 2+ minutes for alert
   # Check Prometheus: http://localhost:9090/alerts
   ```

2. **Trigger High Error Rate**:
   ```bash
   # Simulate API errors (requires exporter modification for testing)
   # Or temporarily misconfigure credentials
   export TADO_TOKEN_PASSPHRASE="wrong-passphrase"
   docker restart tado-exporter
   ```

3. **Check Alert Status**:
   ```bash
   # Query Prometheus
   curl 'http://localhost:9090/api/v1/alerts'

   # Or visit UI
   # http://localhost:9090/alerts
   ```

### Prometheus Expression Editor

Use Prometheus UI (http://localhost:9090) to test alert conditions:

```promql
# Test if exporter is down
up{job="tado-exporter"} == 0

# Test error rate
rate(tado_exporter_scrape_errors_total[5m])

# Test authentication status
tado_exporter_authentication_valid
```

## Customizing Alerts

### Adjusting Thresholds

Edit `tado-exporter.yml` to change alert conditions:

```yaml
# Example: Change high latency threshold from 5s to 10s
- alert: TadoExporterHighScrapeLa tency
  expr: histogram_quantile(0.95, tado_exporter_scrape_duration_seconds) > 10  # Changed from 5
  for: 10m
```

### Adjusting Alert Duration

Change the `for` duration to delay alert firing:

```yaml
# Wait longer before alerting
for: 15m  # Changed from 5m

# Alert faster (less noise, but may catch transients)
for: 1m
```

### Disabling Alerts

Comment out or remove alert rule to disable:

```yaml
# Disabled for testing
# - alert: TadoExporterHighScrapeLa tency
#   expr: ...
```

### Adding Custom Receivers

For new alert channel, add to alertmanager config:

```yaml
receivers:
  - name: 'my-custom-receiver'
    webhook_configs:
      - url: 'http://example.com/webhook'
```

Then add route:

```yaml
routes:
  - match:
      alertname: MyCustomAlert
    receiver: 'my-custom-receiver'
```

## Understanding Alert Labels

Each alert has labels for routing and grouping:

```yaml
labels:
  severity: critical|warning|info
  component: exporter|authentication|collection|performance|...
```

Use these labels to route alerts appropriately:

```yaml
routes:
  # Critical alerts go to PagerDuty
  - match:
      severity: critical
    receiver: pagerduty

  # Authentication alerts to auth team
  - match:
      component: authentication
    receiver: auth-team

  # Performance alerts to ops
  - match:
      component: performance
    receiver: ops-team
```

## Alert Annotations

Each alert includes helpful annotations:

- **`summary`**: Brief description of what's wrong
- **`description`**: Detailed explanation with {{ values }}
- **`runbook`**: Link to troubleshooting guide

Example use in notification template:

```
Alert: {{ .GroupLabels.alertname }}
Severity: {{ .Labels.severity }}
Component: {{ .Labels.component }}

{{ .Alerts.Firing | len }} firing

Runbook: {{ .Alerts.Firing | first | .Annotations.runbook }}
```

## Monitoring Alerts Themselves

Monitor the Prometheus instance that evaluates alerts:

```yaml
# Add meta-alert for Prometheus down
- alert: PrometheusDown
  expr: up{job="prometheus"} == 0
  for: 2m
```

## Best Practices

1. **Set Runbooks**: Always provide runbook URLs in annotations
2. **Group by Context**: Group alerts by home_id, zone_id for easier correlation
3. **Start Conservative**: Use longer `for` durations to avoid alert fatigue
4. **Document Changes**: Keep changelog of alert modifications
5. **Regular Reviews**: Quarterly review alerts for relevance and accuracy
6. **Test Notifications**: Verify notification channels work correctly
7. **Escalation Paths**: Define clear escalation procedures per severity

## Troubleshooting Alerts

### Alerts Not Firing

1. Check alert rule syntax:
   ```bash
   curl http://localhost:9090/api/v1/rules | jq .
   ```

2. Verify metrics exist:
   ```bash
   curl http://localhost:9090/api/v1/query?query=tado_exporter_scrape_errors_total
   ```

3. Check Prometheus logs:
   ```bash
   docker logs prometheus | grep -i alert
   ```

### Alerts Firing but No Notifications

1. Check Alertmanager status:
   ```bash
   curl http://localhost:9093/api/v1/status
   ```

2. Check receiver configuration:
   ```bash
   curl http://localhost:9093/api/v1/alerts
   ```

3. Test receiver manually:
   ```bash
   # For Slack
   curl -X POST -H 'Content-type: application/json' \
     --data '{"text":"Test alert"}' \
     YOUR_SLACK_WEBHOOK_URL
   ```

### Too Many Alerts (Alert Fatigue)

1. Increase `for` duration to filter transient issues
2. Increase alert thresholds to be less sensitive
3. Route low-severity alerts to separate channel
4. Disable info-level alerts if not actionable

## References

- [Prometheus Alerting Documentation](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Alert Best Practices](https://prometheus.io/docs/practices/alerting/)
- [Runbook Template](https://runbooks.cloudd.io/)

## Support

For issues with alerts:
1. Check `RUNBOOK.md` for troubleshooting
2. Review Prometheus/Alertmanager logs
3. Test alert expressions in Prometheus UI
4. File GitHub issue if bug suspected
