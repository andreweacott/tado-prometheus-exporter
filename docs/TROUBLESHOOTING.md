# Troubleshooting Guide

Common issues, diagnostics, and solutions for tado-prometheus-exporter.

## Table of Contents

1. [Authentication Issues](#authentication-issues)
2. [Connection Problems](#connection-problems)
3. [Metrics Collection Issues](#metrics-collection-issues)
4. [Docker/Container Issues](#dockercontainer-issues)
5. [Performance Issues](#performance-issues)
6. [Security Issues](#security-issues)
7. [Alert Troubleshooting](#alert-troubleshooting)
8. [Advanced Debugging](#advanced-debugging)

---

## Authentication Issues

### Device Code Flow Never Completes

**Symptom**: Exporter waits for device code authorization indefinitely

**Causes**:
- User didn't visit authorization URL
- Browser tabs closed before completing
- 5-minute timeout expired

**Solutions**:

1. Check logs for verification URL:
   ```bash
   # Standalone
   grep "verification URL" exporter.log

   # Docker
   docker logs tado-exporter | grep "verification URL"
   ```

2. Visit the URL manually within 5 minutes

3. If timeout occurred, restart:
   ```bash
   pkill tado-prometheus-exporter
   # or
   docker restart tado-exporter
   ```

### Token File Corrupted or Invalid

**Symptom**: Exporter fails with token corruption error

**Causes**:
- Wrong passphrase used
- Token file manually edited
- Disk corruption

**Solutions**:

1. Verify passphrase is correct:
   ```bash
   echo $TADO_TOKEN_PASSPHRASE
   ```

2. Check file permissions:
   ```bash
   ls -la ~/.tado-exporter/token.json
   # Should be: -rw------- (600)
   ```

3. Fix permissions if needed:
   ```bash
   chmod 600 ~/.tado-exporter/token.json
   ```

4. Delete and re-authenticate:
   ```bash
   rm ~/.tado-exporter/token.json
   # Restart exporter - will trigger device code flow
   ```

### Passphrase Required Error

**Symptom**: Error indicates passphrase is missing or empty

**Causes**:
- `TADO_TOKEN_PASSPHRASE` environment variable not set
- Empty passphrase passed

**Solutions**:

1. Set environment variable:
   ```bash
   export TADO_TOKEN_PASSPHRASE="my-secure-passphrase"
   ```

2. Verify it's set:
   ```bash
   echo $TADO_TOKEN_PASSPHRASE
   ```

3. For Docker, check environment variable:
   ```bash
   docker inspect tado-exporter | grep TADO_TOKEN_PASSPHRASE
   ```

---

## Connection Problems

### Cannot Connect to Tado API

**Symptom**: Connection refused or timeout reaching Tado API

**Causes**:
- Network connectivity issue
- Firewall blocking outbound connections
- Tado API is down
- DNS resolution failing

**Solutions**:

1. Test network connectivity:
   ```bash
   # Test Tado API domain
   ping api.tado.com
   nslookup api.tado.com
   curl -I https://api.tado.com
   ```

2. Check firewall rules:
   ```bash
   # On Linux with ufw
   sudo ufw status
   sudo ufw allow out to any port 443
   ```

3. Verify DNS resolution:
   ```bash
   nslookup my.tado.com
   # Should resolve to an IP address
   ```

4. Check Tado API status:
   - Visit: https://status.tado.com
   - Look for incidents or maintenance

### Connection Timeout (10s default)

**Symptom**: Exporter times out when collecting metrics

**Causes**:
- API is slow or unresponsive
- Home has many zones (takes time to fetch all)
- Network latency is high

**Solutions**:

1. Increase scrape timeout:
   ```bash
   # Command-line
   ./exporter --scrape-timeout 30

   # Environment variable
   TADO_SCRAPE_TIMEOUT=30

   # Docker
   docker run -e TADO_SCRAPE_TIMEOUT=30 ...
   ```

2. Check how many zones you have:
   ```bash
   curl -s http://localhost:9100/metrics | grep zone_name | sort -u | wc -l
   ```

3. Check network latency:
   ```bash
   ping api.tado.com
   # Look for high latency values
   ```

---

## Metrics Collection Issues

### No Metrics Exposed / Empty /metrics Endpoint

**Symptom**: `/metrics` endpoint returns empty or minimal metrics

**Causes**:
- Authentication not completed
- Token invalid
- No zones in home
- Exporter crashed during collection

**Solutions**:

1. Check exporter is running:
   ```bash
   curl http://localhost:9100/health
   # Should return: {"status":"ok",...}
   ```

2. Check auth completed:
   ```bash
   ls -la ~/.tado-exporter/token.json
   # File should exist
   ```

3. Verify metrics endpoint:
   ```bash
   curl http://localhost:9100/metrics | head -20
   # Should see TYPE and HELP comments
   ```

4. Check logs for errors:
   ```bash
   # Enable debug logging
   TADO_LOG_LEVEL=debug ./exporter ...

   # Or for Docker
   docker logs tado-exporter | tail -50
   ```

5. Verify home has zones:
   ```bash
   # Log into my.tado.com and verify you have rooms/zones
   ```

### Some Metrics Missing

**Symptom**: Only partial metrics are exposed

**Causes**:
- Some zones are offline
- Tado device not fully provisioned
- API returned partial data

**Solutions**:

1. Check which metrics are missing:
   ```bash
   # List all metric names
   curl http://localhost:9100/metrics | grep "^tado_" | cut -d'{' -f1 | sort -u
   ```

2. Check home data:
   ```bash
   # Log into my.tado.com
   # Verify:
   # - Resident presence settings enabled
   # - Solar intensity available in your region
   # - All zones are online
   ```

### Metric Values Are Old/Stale

**Symptom**: Metrics show outdated values

**Causes**:
- Prometheus cache interval is too long
- Exporter is not being scraped
- Zone is offline

**Solutions**:

1. Check Prometheus scrape interval:
   ```yaml
   scrape_configs:
     - job_name: 'tado-exporter'
       scrape_interval: 30s  # Reduce if too long
       static_configs:
         - targets: ['localhost:9100']
   ```

2. Verify Prometheus is scraping:
   ```bash
   # In Prometheus UI
   # - Targets page should show exporter as "UP"
   # - Check last scrape time
   ```

---

## Docker/Container Issues

### Container Exits Immediately

**Symptom**: Docker container starts then stops

**Causes**:
- Missing environment variable
- Invalid argument
- Token file permission issue

**Solutions**:

1. Check logs:
   ```bash
   docker logs tado-exporter
   # Look for error messages
   ```

2. Run with interactive mode:
   ```bash
   docker run -it --rm \
     -e TADO_TOKEN_PASSPHRASE="passphrase" \
     tado-prometheus-exporter:test
   ```

3. Verify environment variables:
   ```bash
   docker run --rm \
     -e TADO_TOKEN_PASSPHRASE="passphrase" \
     tado-prometheus-exporter:test \
     env | grep TADO
   ```

### Container Health Check Failing

**Symptom**: Docker shows "unhealthy" status

**Causes**:
- Health endpoint not responding
- Metrics collection taking too long
- Container out of memory

**Solutions**:

1. Check health status:
   ```bash
   docker inspect tado-exporter --format='{{.State.Health}}'
   ```

2. Test health endpoint manually:
   ```bash
   docker exec tado-exporter wget -O- http://localhost:9100/health
   ```

3. Check resource usage:
   ```bash
   docker stats tado-exporter
   # If memory/CPU at limit, increase them
   ```

### Can't Access Metrics from Host

**Symptom**: Cannot reach `localhost:9100/metrics` from host

**Causes**:
- Port not forwarded
- Firewall blocking
- Container using wrong network

**Solutions**:

1. Verify port mapping:
   ```bash
   docker ps | grep tado-exporter
   # Look for: 0.0.0.0:9100->9100/tcp
   ```

2. Test from container:
   ```bash
   docker exec tado-exporter curl http://localhost:9100/metrics | head
   # Should work if inside container
   ```

3. Check if port is in use:
   ```bash
   netstat -tlnp | grep 9100
   lsof -i :9100
   ```

---

## Performance Issues

### High CPU Usage

**Symptom**: Exporter consuming too much CPU

**Causes**:
- Too many zones (many API calls)
- Prometheus scraping too frequently
- Logging level set to debug

**Solutions**:

1. Reduce Prometheus scrape frequency:
   ```yaml
   scrape_configs:
     - job_name: 'tado-exporter'
       scrape_interval: 120s  # Increase interval
       static_configs:
         - targets: ['localhost:9100']
   ```

2. Check log level:
   ```bash
   # Set to 'warn' or 'error' in production
   TADO_LOG_LEVEL=warn ./exporter ...
   ```

3. Limit CPU in Docker:
   ```yaml
   services:
     exporter:
       cpus: '0.5'  # Limit to 50% of one CPU
   ```

### High Memory Usage

**Symptom**: Exporter using lots of memory

**Causes**:
- Memory leak (unlikely in Go)
- Large number of metrics buffered

**Solutions**:

1. Monitor memory usage:
   ```bash
   docker stats tado-exporter
   # or
   ps aux | grep exporter
   ```

2. Set memory limit:
   ```yaml
   services:
     exporter:
       mem_limit: 512m
   ```

### Slow Metrics Collection

**Symptom**: Takes >5 seconds to collect all metrics

**Causes**:
- Many zones (N zones = N API calls)
- Slow network
- Tado API is slow

**Solutions**:

1. Measure collection time:
   ```bash
   time curl -s http://localhost:9100/metrics > /dev/null
   ```

2. Check zone count:
   ```bash
   curl http://localhost:9100/metrics | grep -c zone_name
   ```

3. Increase scrape timeout:
   ```bash
   TADO_SCRAPE_TIMEOUT=30 ./exporter ...
   ```

---

## Security Issues

### Token File Not Encrypted or Exposed

**Symptom**: Token file is readable in plain text

**Causes**:
- Wrong file permissions
- Passphrase not set

**Solutions**:

1. Verify file permissions:
   ```bash
   ls -la ~/.tado-exporter/token.json
   # Should be: -rw------- (600)
   ```

2. Fix permissions:
   ```bash
   chmod 600 ~/.tado-exporter/token.json
   ```

3. If actually exposed, regenerate:
   ```bash
   rm ~/.tado-exporter/token.json
   # Restart exporter to re-authenticate
   ```

### Passphrase in Command History

**Solutions**:

1. Use environment file instead:
   ```bash
   # Create file with restricted permissions
   cat > ~/.tado-exporter/env << EOF
   TADO_TOKEN_PASSPHRASE=your-passphrase
   EOF
   chmod 600 ~/.tado-exporter/env

   # Source it
   source ~/.tado-exporter/env
   ```

---

## Alert Troubleshooting

### Exporter Down

**What it means**: Exporter unreachable for 2+ minutes

**Troubleshooting**:

1. Check if running:
   ```bash
   ps aux | grep tado-exporter
   docker ps | grep tado-exporter
   ```

2. Restart:
   ```bash
   docker restart tado-exporter
   ```

3. Check logs:
   ```bash
   docker logs tado-exporter | tail -50
   ```

### High Error Rate

**What it means**: More than 10% of collection attempts failing

**Troubleshooting**:

1. Check current errors:
   ```bash
   curl http://localhost:9100/metrics | grep scrape_errors
   ```

2. Check logs:
   ```bash
   docker logs tado-exporter --since 5m | grep -i error
   ```

3. Verify Tado API connectivity:
   ```bash
   curl https://api.tado.com
   ```

### Authentication Invalid

**What it means**: Exporter cannot authenticate with Tado API

**Troubleshooting**:

1. Check token file:
   ```bash
   ls -la ~/.tado-exporter/token.json
   ```

2. Verify passphrase:
   ```bash
   echo $TADO_TOKEN_PASSPHRASE
   ```

3. Check logs:
   ```bash
   docker logs tado-exporter | grep -i auth
   ```

4. Regenerate token:
   ```bash
   rm ~/.tado-exporter/token.json
   docker restart tado-exporter
   ```

### Missing Metrics

**What it means**: No temperature metrics for 10+ minutes

**Troubleshooting**:

1. Check if exporter has data:
   ```bash
   curl http://localhost:9100/metrics | grep tado_temperature
   ```

2. Verify zones exist:
   ```bash
   curl http://localhost:9100/metrics | grep zone_name | head
   ```

3. Check logs:
   ```bash
   docker logs tado-exporter | tail -20
   ```

4. Wait for first scrape:
   ```bash
   # Check scrape count (should be > 0)
   curl http://localhost:9100/metrics | grep scrape_duration_seconds_count
   ```

---

## Advanced Debugging

### Enable Debug Logging

```bash
# Maximum verbosity
TADO_LOG_LEVEL=debug ./exporter ... 2>&1 | tee exporter.log

# For Docker
docker run -e TADO_LOG_LEVEL=debug tado-prometheus-exporter:test
```

### Check Metrics in Detail

```bash
# Get raw metrics with all labels
curl http://localhost:9100/metrics | grep "tado_"

# Search for specific zone
curl http://localhost:9100/metrics | grep 'zone_name="Bedroom"'

# Count metrics by type
curl http://localhost:9100/metrics | grep -v "^#" | grep "^tado_" | cut -d'{' -f1 | sort | uniq -c
```

### Check System Resources

```bash
# Overall system status
free -h  # Memory
df -h    # Disk space
top      # CPU and memory

# Exporter-specific process
ps aux | grep exporter
```

### Network Diagnostics

```bash
# Test HTTP requests
curl -v http://localhost:9100/metrics 2>&1 | head -30

# Check DNS resolution
dig api.tado.com
nslookup api.tado.com
```

---

## Getting Help

If troubleshooting doesn't resolve your issue:

1. **Collect diagnostic information:**
   ```bash
   mkdir diagnostics
   echo "=== Configuration ===" > diagnostics/info.txt
   env | grep TADO >> diagnostics/info.txt

   echo "=== Metrics ===" > diagnostics/metrics.txt
   curl http://localhost:9100/metrics >> diagnostics/metrics.txt 2>&1

   echo "=== Logs ===" > diagnostics/logs.txt
   docker logs tado-exporter >> diagnostics/logs.txt
   ```

2. **Open GitHub Issue:**
   - Include diagnostic information
   - Describe what you've tried
   - Include error messages and logs
   - Don't include passphrase or tokens

3. **Check existing issues:**
   - https://github.com/andreweacott/tado-prometheus-exporter/issues

---

## Related Documentation

- [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment options
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [HTTP_ENDPOINTS.md](HTTP_ENDPOINTS.md) - API reference
- [README.md](../README.md) - Quick start
