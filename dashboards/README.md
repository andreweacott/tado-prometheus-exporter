# Grafana Dashboards - Tado Exporter

This directory contains pre-built Grafana dashboards for monitoring Tado smart heating systems via the Prometheus exporter.

## Dashboard Overview

### `tado-exporter.json` - Comprehensive Monitoring Dashboard

A complete monitoring dashboard for the Tado Prometheus exporter with real-time visualization and historical trends.

**Features**:
- **Authentication Status**: Live authentication health indicator
- **Temperature Monitoring**: 24-hour temperature trends by zone with mean/max/min
- **Humidity Monitoring**: Relative humidity levels across zones
- **Heating Power**: Real-time heating system power output by zone
- **Exporter Health**:
  - Scrape error count (5-minute window)
  - P95 scrape latency
  - Scrape attempt count
- **Weather Data**:
  - Outside temperature trends
  - Solar intensity patterns
  - Resident presence tracking

**Time Range**: Last 24 hours (configurable)

**Refresh Rate**: 30 seconds (recommended)

## Installation Instructions

### Option 1: Manual Import via Grafana UI

1. **Open Grafana**:
   - Navigate to your Grafana instance (usually `http://localhost:3000`)
   - Log in with your admin credentials

2. **Import Dashboard**:
   - Click the **+** icon in the left sidebar
   - Select **Import**
   - Choose one of the following:
     - **Upload JSON file**: Click "Upload JSON file" and select `tado-exporter.json`
     - **Paste JSON**: Copy the contents of `tado-exporter.json` and paste into the text area

3. **Configure Datasource**:
   - If prompted, select your Prometheus datasource
   - If not prompted, the dashboard will use the default Prometheus datasource

4. **Import**:
   - Click **Import**
   - The dashboard will appear and start showing data

### Option 2: Programmatic Import via HTTP API

```bash
curl -X POST \
  http://localhost:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <YOUR_GRAFANA_API_TOKEN>' \
  -d @tado-exporter.json
```

Replace `<YOUR_GRAFANA_API_TOKEN>` with your Grafana API token.

### Option 3: Kubernetes Configmap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards
  namespace: monitoring
data:
  tado-exporter.json: |
    <contents of tado-exporter.json>
---
apiVersion: v1
kind: Pod
metadata:
  name: grafana
  namespace: monitoring
spec:
  containers:
  - name: grafana
    image: grafana/grafana:latest
    volumeMounts:
    - name: dashboards
      mountPath: /etc/grafana/provisioning/dashboards
  volumes:
  - name: dashboards
    configMap:
      name: grafana-dashboards
```

## Dashboard Customization

### Modifying Time Range

Edit the dashboard and change the global time range:
1. Click the time picker (top-right, default "Last 24 hours")
2. Select a new range or enter a custom range

### Modifying Refresh Rate

1. Click the refresh icon (top-right)
2. Select a new interval or set a custom value
3. Click **Apply**

### Adding New Panels

To add additional panels to monitor specific metrics:

1. Click **Add panel** (+ icon)
2. Select **Prometheus** as the datasource
3. Enter a Prometheus query (examples below)
4. Configure visualization and save

#### Example Queries

**Zone Window Status** (shows if windows are open):
```
tado_is_window_open{zone_name="Living Room"}
```

**Zone Power Status** (shows if zones are on/off):
```
tado_is_zone_powered{zone_name="Bedroom"}
```

**Temperature Set vs Measured**:
```
{__name__=~"tado_temperature_(measured|set)_celsius", zone_name="Living Room"}
```

**Scrape Duration Distribution**:
```
histogram_quantile(0.95, tado_exporter_scrape_duration_seconds)
```

## Metrics Reference

### Zone Metrics

- `tado_temperature_measured_celsius` - Current room temperature
- `tado_temperature_measured_fahrenheit` - Current room temperature (°F)
- `tado_temperature_set_celsius` - Target/setpoint temperature
- `tado_temperature_set_fahrenheit` - Target/setpoint temperature (°F)
- `tado_humidity_measured_percentage` - Relative humidity (0-100%)
- `tado_heating_power_percentage` - Heating system power output (0-100%)
- `tado_is_window_open` - Window open detection (1=open, 0=closed)
- `tado_is_zone_powered` - Zone power state (1=on, 0=off)

**Labels**: `home_id`, `zone_id`, `zone_name`, `zone_type`

### Home Metrics

- `tado_is_resident_present` - Resident presence status (1=home, 0=away)
- `tado_temperature_outside_celsius` - Outside temperature
- `tado_temperature_outside_fahrenheit` - Outside temperature (°F)
- `tado_solar_intensity_percentage` - Solar radiation intensity (0-100%)

### Exporter Health Metrics

- `tado_exporter_scrape_duration_seconds` - Metric collection duration (histogram)
- `tado_exporter_scrape_errors_total` - Collection error counter
- `tado_exporter_authentication_valid` - Auth status (1=valid, 0=invalid)
- `tado_exporter_authentication_errors_total` - Auth error counter
- `tado_exporter_last_authentication_success_unix` - Last successful auth timestamp
- `tado_exporter_build_info` - Build information (always 1)

## Troubleshooting

### Dashboard Shows "No Data"

**Possible causes**:
1. Prometheus datasource not configured
2. Exporter not running or not scraping metrics
3. Prometheus not scraping the exporter endpoint
4. Metrics don't exist yet (wait 1-2 minutes for first scrape)

**Solution**:
1. Verify Prometheus can reach the exporter: `curl http://localhost:9100/metrics`
2. Check Prometheus targets: http://localhost:9090/targets
3. Verify the dashboard datasource is set correctly (Edit > Datasource)

### Panels Show Blank Graphs

**Possible causes**:
1. No data in the time range selected
2. Query syntax error
3. Metric labels don't match actual data

**Solution**:
1. Expand time range to 7d or 30d
2. Check the query in the Query Editor (Edit panel > Queries tab)
3. Use Prometheus Query Browser to test: http://localhost:9090

### "Templating Error"

If you see templating errors, it's likely variables were misconfigured.

**Solution**:
1. Edit the dashboard
2. Click Settings (gear icon)
3. Go to Variables
4. Delete any broken variables
5. Save the dashboard

## Version Compatibility

| Component | Version | Status |
|-----------|---------|--------|
| Grafana | 8.0+ | Tested |
| Prometheus | 2.30+ | Tested |
| Tado Exporter | 1.0+ | Tested |

## Support

For issues with the dashboard:
1. Check the Grafana logs: `docker logs <grafana-container>`
2. Check Prometheus targets are healthy
3. Verify the exporter is running: `curl http://localhost:9100/health`

## Tips & Best Practices

### Setting Up Alerts

Use the dashboard metrics to create Prometheus alerts:

```yaml
groups:
  - name: tado
    rules:
      - alert: TadoExporterDown
        expr: up{job="tado-exporter"} == 0
        for: 5m

      - alert: TadoTemperatureHigh
        expr: tado_temperature_measured_celsius > 25
        for: 10m

      - alert: TadoAuthenticationFailed
        expr: tado_exporter_authentication_valid == 0
        for: 5m
```

### Multi-Home Monitoring

To monitor multiple homes, use the dashboard filters:

1. Edit the dashboard
2. Add a variable for home filtering
3. Use the variable in queries: `{home_id="$home_id"}`

### Custom Dashboards

To create your own dashboard:

1. Start with this dashboard as a template
2. Add/remove panels as needed
3. Export as JSON: Dashboard > Share > Export
4. Share with team or version control

## References

- [Tado Exporter GitHub](https://github.com/andreweacott/tado-prometheus-exporter)
- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
