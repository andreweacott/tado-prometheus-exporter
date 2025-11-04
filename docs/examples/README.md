# Prometheus Alerting Rules - Tado Exporter

Example Prometheus alert rules for monitoring the Tado exporter are included in [tado-exporter-rules.yml](./tado-exporter-rules.yml)

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
