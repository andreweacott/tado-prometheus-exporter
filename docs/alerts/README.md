# Prometheus Alerting Rules - Tado Exporter

Pre-configured Prometheus alert rules for monitoring the Tado exporter in homelab environments.

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

---

## Alert Rules Overview

### Critical Alerts

| Alert | Condition | Action |
|-------|-----------|--------|
| `TadoExporterDown` | Unreachable for 2m | Restart exporter / check infrastructure |
| `TadoExporterAuthenticationInvalid` | Invalid auth for 5m | Regenerate token or check credentials |
| `TadoExporterHighScrapingErrorRate` | >10% errors for 10m | Check API connectivity or auth |
| `TadoAPIUnreachable` | >50% failures for 5m | Check API status and network |

### Warning Alerts

| Alert | Condition | Action |
|-------|-----------|--------|
| `TadoExporterAuthenticationFailures` | Auth errors in 5m | Monitor, may be transient |
| `TadoExporterAuthenticationStale` | No auth for 1+ hours | Check if exporter is running |
| `TadoExporterHighScrapeLa tency` | P95 latency >5s for 10m | Monitor API performance |
| `TadoExporterScrapingErrors` | Collection failures in 5m | Check logs for details |
| `TadoExporterMissingMetrics` | No temp metrics for 10m | Wait for first scrape or check config |

---

## Alertmanager Setup

### Minimal Configuration

Create `/etc/alertmanager/alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: 'default'
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

receivers:
  - name: 'default'
    # Add your notification channel here
```

### Slack Integration

```yaml
global:
  resolve_timeout: 5m
  slack_api_url: 'YOUR_SLACK_WEBHOOK_URL'

route:
  receiver: 'slack'
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

receivers:
  - name: 'slack'
    slack_configs:
      - channel: '#monitoring'
        title: 'Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts.Firing }}{{ .Annotations.description }}{{ end }}'
```

### Email Integration

```yaml
receivers:
  - name: 'email'
    email_configs:
      - to: 'your-email@example.com'
        from: 'prometheus@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'prometheus'
        auth_password: 'password'
```

---

## Customizing Alerts

### Adjusting Thresholds

Edit `tado-exporter.yml` to change alert conditions:

```yaml
# Example: Change high latency threshold from 5s to 10s
- alert: TadoExporterHighScrapeLa tency
  expr: histogram_quantile(0.95, tado_exporter_scrape_duration_seconds) > 10
  for: 10m
```

### Adjusting Duration

Change the `for` duration to delay alert firing:

```yaml
# Wait longer before alerting (reduces false positives)
for: 15m

# Alert faster (catches issues sooner, more noise)
for: 1m
```

### Disabling Alerts

Comment out alert rules to disable:

```yaml
# Disabled for testing
# - alert: TadoExporterHighScrapeLa tency
#   expr: ...
```

---

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
   # Misconfigure credentials
   export TADO_TOKEN_PASSPHRASE="wrong-passphrase"
   docker restart tado-exporter
   ```

### Check Alert Status

```bash
# Query Prometheus
curl 'http://localhost:9090/api/v1/alerts'

# Or visit UI
# http://localhost:9090/alerts
```

### Test Alert Expressions

Use Prometheus UI (http://localhost:9090) to test alert conditions:

```promql
# Test if exporter is down
up{job="tado-exporter"} == 0

# Test error rate
rate(tado_exporter_scrape_errors_total[5m])

# Test authentication status
tado_exporter_authentication_valid
```

---

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

2. Test notification channel manually:
   ```bash
   # For Slack
   curl -X POST -H 'Content-type: application/json' \
     --data '{"text":"Test alert"}' \
     YOUR_SLACK_WEBHOOK_URL
   ```

### Too Many Alerts (Alert Fatigue)

1. Increase `for` duration to filter transient issues
2. Increase alert thresholds to be less sensitive
3. Disable low-priority alerts

---

## Best Practices

1. **Start Conservative**: Use longer `for` durations to avoid alert fatigue
2. **Monitor Trends**: Check Prometheus for trending metrics over time
3. **Test Regularly**: Verify alerts work by testing notification channels
4. **Document Changes**: Keep track of alert modifications
5. **Review Quarterly**: Check if alerts are still relevant

---

## References

- [Prometheus Alerting Documentation](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Alert Best Practices](https://prometheus.io/docs/practices/alerting/)
- [tado-exporter.yml](tado-exporter.yml) - Actual alert rules
- [TROUBLESHOOTING.md](../TROUBLESHOOTING.md) - Runbook for each alert

---

## Support

For issues with alerts:
1. Check [TROUBLESHOOTING.md](../TROUBLESHOOTING.md) for runbook details
2. Review Prometheus/Alertmanager logs
3. Test alert expressions in Prometheus UI
4. File GitHub issue if bug suspected: https://github.com/andreweacott/tado-prometheus-exporter/issues
