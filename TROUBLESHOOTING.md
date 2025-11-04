# Troubleshooting Guide

Common issues, diagnostics, and solutions for tado-prometheus-exporter.

## Table of Contents

1. [Authentication Issues](#authentication-issues)
2. [Connection Problems](#connection-problems)
3. [Metrics Collection Issues](#metrics-collection-issues)
4. [Docker/Container Issues](#dockercontainer-issues)
5. [Performance Issues](#performance-issues)
6. [Security Issues](#security-issues)
7. [Advanced Debugging](#advanced-debugging)

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

2. Visit the URL manually within 5 minutes:
   ```
   https://my.tado.com/device-code?userCode=XXXX-XXXX
   ```

3. If timeout occurred, restart:
   ```bash
   # Kill current process
   pkill tado-prometheus-exporter

   # Restart
   ./exporter --token-path ~/.tado-exporter/token.json \
              --token-passphrase "$TADO_TOKEN_PASSPHRASE"
   ```

### "Token file corrupted or invalid"

**Symptom**: Exporter fails with token corruption error

**Causes**:
- Wrong passphrase used
- Token file manually edited
- Disk corruption
- File permissions issue

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

### "Passphrase required" Error

**Symptom**: Error indicates passphrase is missing or empty

**Causes**:
- `TADO_TOKEN_PASSPHRASE` environment variable not set
- Empty passphrase passed
- Typo in variable name

**Solutions**:

1. Set environment variable:
   ```bash
   export TADO_TOKEN_PASSPHRASE="my-secure-passphrase"
   ```

2. Verify it's set:
   ```bash
   echo $TADO_TOKEN_PASSPHRASE
   # Should show your passphrase, not empty
   ```

3. For systemd, check `/etc/default/tado-exporter`:
   ```bash
   sudo cat /etc/default/tado-exporter
   # Should have: TADO_TOKEN_PASSPHRASE=<passphrase>
   ```

4. For Docker, check environment variable:
   ```bash
   docker inspect tado-exporter | grep TADO_TOKEN_PASSPHRASE
   ```

---

## Connection Problems

### "Connection refused" to Tado API

**Symptom**: Cannot reach Tado API, connection refused

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

   # Check if HTTPS is blocked
   sudo netstat -tlnp | grep -i listen
   ```

3. Test from container:
   ```bash
   docker run --rm tado-prometheus-exporter:test \
     curl -I https://api.tado.com
   ```

4. Verify DNS resolution:
   ```bash
   nslookup my.tado.com
   # Should resolve to an IP address
   ```

5. Check Tado API status:
   - Visit: https://status.tado.com
   - Look for incidents or maintenance

### "Connection timed out" (10s timeout)

**Symptom**: Exporter times out when collecting metrics

**Causes**:
- API is slow or unresponsive
- Home has many zones (takes time to fetch all)
- Network latency is high
- API rate limiting

**Solutions**:

1. Increase scrape timeout:
   ```bash
   # Command-line
   --scrape-timeout 30

   # Environment variable
   TADO_SCRAPE_TIMEOUT=30

   # Docker
   docker run -e TADO_SCRAPE_TIMEOUT=30 ...
   ```

2. Check how many zones you have:
   ```bash
   # The exporter fetches each zone's data individually
   # More zones = longer fetch time
   curl -s http://localhost:9100/metrics | grep zone_name | sort -u | wc -l
   ```

3. Check API response times:
   ```bash
   time curl -s https://api.tado.com/api/v2/homes/$(home_id)/zones | jq . > /dev/null
   # Look for how long it takes
   ```

4. Check network latency:
   ```bash
   ping api.tado.com
   # Look for high latency values
   ```

### "No route to host" Error

**Symptom**: Cannot reach Tado API servers

**Causes**:
- No internet connection
- Network interface down
- Routing misconfigured
- Geolocking by ISP

**Solutions**:

1. Check network status:
   ```bash
   # Check if connected to network
   ip link show
   ip addr show

   # Check default route
   ip route show
   ```

2. Test connectivity to public DNS:
   ```bash
   # Try Google DNS
   ping 8.8.8.8

   # Try Cloudflare DNS
   ping 1.1.1.1
   ```

3. For Docker, check network:
   ```bash
   # Check if container can reach outside
   docker exec tado-exporter ping 8.8.8.8

   # Inspect network settings
   docker inspect tado-exporter --format='{{json .NetworkSettings}}'
   ```

4. Check ISP/firewall geolocking:
   ```bash
   # Try VPN if available (not recommended in production)
   # Contact ISP if port 443 is blocked
   ```

---

## Metrics Collection Issues

### "No metrics exposed" / Empty /metrics endpoint

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
   # Should return: {"status":"healthy",...}
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
   docker exec tado-exporter TADO_LOG_LEVEL=debug
   ```

5. Verify home has zones:
   ```bash
   # Check if zones exist in your Tado home
   # Log into my.tado.com and verify you have rooms/zones
   ```

### "Some metrics missing"

**Symptom**: Only partial metrics are exposed

**Causes**:
- Some zones are offline
- Tado device not fully provisioned
- API returned partial data
- Specific metric type not available

**Solutions**:

1. Check which metrics are missing:
   ```bash
   # List all metric names
   curl http://localhost:9100/metrics | grep "^tado_" | cut -d'{' -f1 | sort -u

   # Should see:
   # tado_is_resident_present
   # tado_solar_intensity_percentage
   # tado_temperature_measured_celsius
   # ... etc
   ```

2. Check home data:
   ```bash
   # Log into my.tado.com
   # Verify:
   # - Resident presence settings enabled
   # - Solar intensity available in your region
   # - All zones are online
   ```

3. Enable debug logs to see what's collected:
   ```bash
   TADO_LOG_LEVEL=debug ./exporter ... 2>&1 | grep "metric"
   ```

### "Metric values are old/stale"

**Symptom**: Metrics show outdated values

**Causes**:
- Prometheus cache interval is too long
- Exporter is not being scraped
- API is returning cached data
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

3. Check exporter is collecting new data:
   ```bash
   # Fetch metrics twice with delay
   curl http://localhost:9100/metrics | grep "tado_temperature" | head -3
   sleep 5
   curl http://localhost:9100/metrics | grep "tado_temperature" | head -3
   # Timestamp in HELP should differ (if available)
   ```

4. Check if zone is offline:
   ```bash
   curl http://localhost:9100/metrics | grep "zone_name"
   # If zone is offline, values might not update
   ```

---

## Docker/Container Issues

### Container exits immediately

**Symptom**: Docker container starts then stops

**Causes**:
- Missing environment variable
- Invalid argument
- Token file permission issue
- Crash during startup

**Solutions**:

1. Check logs:
   ```bash
   docker logs tado-exporter
   # Look for error messages
   ```

2. Run with interactive mode to see errors:
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

4. Check volume permissions:
   ```bash
   # Create volume with correct permissions
   docker volume rm tado-tokens
   docker volume create tado-tokens
   ```

### Container health check failing

**Symptom**: Docker shows "unhealthy" status

**Causes**:
- Health endpoint not responding
- Metrics collection taking too long
- Container killed due to resource limits

**Solutions**:

1. Check health status:
   ```bash
   docker inspect tado-exporter --format='{{.State.Health}}'
   ```

2. Test health endpoint manually:
   ```bash
   docker exec tado-exporter wget -O- http://localhost:9100/health
   ```

3. Increase health check timeout:
   ```yaml
   # docker-compose.yml
   healthcheck:
     test: ["CMD", "wget", "--timeout=10", "-O-", "http://localhost:9100/health"]
     interval: 30s
     timeout: 10s
     retries: 3
   ```

4. Check resource limits:
   ```bash
   docker stats tado-exporter
   # If memory/CPU at limit, increase them
   ```

### Can't access metrics from host machine

**Symptom**: Cannot reach `localhost:9100/metrics` from host

**Causes**:
- Port not forwarded
- Firewall blocking
- Container using wrong network
- Port already in use

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

3. Check firewall:
   ```bash
   # On Linux
   sudo firewall-cmd --list-ports
   sudo firewall-cmd --add-port=9100/tcp --permanent

   # On macOS (Docker Desktop)
   # Already forwards ports to host
   ```

4. Check if port is in use:
   ```bash
   netstat -tlnp | grep 9100
   lsof -i :9100
   # Kill if something else is using port
   ```

### Docker Compose services won't start

**Symptom**: `docker-compose up` fails

**Causes**:
- Environment variables not set
- Volume already in use
- Network port conflict
- Invalid YAML syntax

**Solutions**:

1. Check `.env` file exists:
   ```bash
   ls -la .env
   cat .env
   # Should have TADO_TOKEN_PASSPHRASE=...
   ```

2. Validate YAML:
   ```bash
   docker-compose config
   # Will show any syntax errors
   ```

3. Clean up and retry:
   ```bash
   docker-compose down -v
   docker-compose up -d
   ```

4. Check port conflicts:
   ```bash
   netstat -tlnp | grep -E "9100|9090|3000"
   # Kill any existing services on these ports
   ```

---

## Performance Issues

### High CPU Usage

**Symptom**: Exporter consuming too much CPU

**Causes**:
- Too many zones (many API calls)
- Prometheus scraping too frequently
- Metrics conversion overhead
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

3. Monitor per-zone collection time:
   ```bash
   TADO_LOG_LEVEL=debug ./exporter ... 2>&1 | grep "zone collection"
   ```

4. Limit CPU in Docker:
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
- Prometheus client buffering

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
       mem_limit: 512m  # Limit to 512MB
   ```

3. Restart periodically:
   ```bash
   # In Docker Compose
   restart: on-failure
   ```

### Slow Metrics Collection

**Symptom**: Takes >5 seconds to collect all metrics

**Causes**:
- Many zones (N zones = N API calls)
- Slow network
- Tado API is slow
- API rate limiting

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

4. Reduce Prometheus scrape frequency:
   ```yaml
   scrape_interval: 120s  # Scrape every 2 minutes instead of 1
   ```

---

## Security Issues

### Token file not encrypted / accidentally exposed

**Symptom**: Token file is readable in plain text

**Causes**:
- Passphrase not set
- File manually decrypted
- Wrong file permissions

**Solutions**:

1. Verify token encryption:
   ```bash
   file ~/.tado-exporter/token.json
   # Should NOT show readable text
   ```

2. Verify file permissions:
   ```bash
   ls -la ~/.tado-exporter/token.json
   # Should be: -rw------- (600) - only readable by owner
   ```

3. Fix permissions:
   ```bash
   chmod 600 ~/.tado-exporter/token.json
   chown $USER:$GROUP ~/.tado-exporter/token.json
   ```

4. If actually exposed, regenerate:
   ```bash
   rm ~/.tado-exporter/token.json
   # Restart exporter to re-authenticate
   ```

### Passphrase in command-line history

**Symptom**: Passphrase visible in bash history

**Solutions**:

1. Clear history:
   ```bash
   history -c
   cat /dev/null > ~/.bash_history
   ```

2. Use environment file instead:
   ```bash
   # Create file with restricted permissions
   cat > ~/.tado-exporter/env << EOF
   TADO_TOKEN_PASSPHRASE=your-passphrase
   EOF
   chmod 600 ~/.tado-exporter/env

   # Source it
   source ~/.tado-exporter/env
   ```

3. For systemd, use environment file:
   ```bash
   # Already done in deployment guide
   sudo cat > /etc/default/tado-exporter <<EOF
   TADO_TOKEN_PASSPHRASE=...
   EOF
   sudo chmod 600 /etc/default/tado-exporter
   ```

### Container running as root

**Symptom**: Security concern - container runs as root

**Solutions** (Docker Compose):

```yaml
services:
  exporter:
    user: 1000:1000  # Run as unprivileged user
    # Or create user in Dockerfile
```

---

## Advanced Debugging

### Enable debug logging

```bash
# Maximum verbosity
TADO_LOG_LEVEL=debug ./exporter ... 2>&1 | tee exporter.log

# For Docker
docker run -e TADO_LOG_LEVEL=debug tado-prometheus-exporter:test

# For Docker Compose
TADO_LOG_LEVEL=debug docker-compose up
```

### Check metrics in detail

```bash
# Get raw metrics with all labels
curl http://localhost:9100/metrics | grep "tado_"

# Search for specific zone
curl http://localhost:9100/metrics | grep 'zone_name="Bedroom"'

# Count metrics by type
curl http://localhost:9100/metrics | grep -v "^#" | grep "^tado_" | cut -d'{' -f1 | sort | uniq -c
```

### Trace API calls (requires code modification)

Add to pkg/collector/collector.go:

```go
log.Debugf("Fetching home data for homeID: %s", tc.homeID)
// ... API call
log.Debugf("Received home data: %+v", home)
```

### Check system resources

```bash
# Overall system status
free -h  # Memory
df -h    # Disk space
top      # CPU and memory

# Exporter-specific process
ps aux | grep exporter
lsof -p $(pgrep exporter)  # Files and sockets
```

### Network diagnostics

```bash
# Trace HTTP requests (requires curl with verbose)
curl -v http://localhost:9100/metrics 2>&1 | head -30

# Monitor network traffic
tcpdump -i any -n 'port 9100 or port 443'

# Check DNS resolution
dig api.tado.com
nslookup api.tado.com
```

---

## Getting Help

If troubleshooting doesn't resolve your issue:

1. **Collect diagnostic information:**
   ```bash
   # Create diagnostic bundle
   mkdir diagnostics
   echo "=== Version ===" > diagnostics/info.txt
   ./exporter --version >> diagnostics/info.txt 2>&1

   echo "=== Configuration ===" >> diagnostics/info.txt
   env | grep TADO >> diagnostics/info.txt

   echo "=== Metrics ===" > diagnostics/metrics.txt
   curl http://localhost:9100/metrics >> diagnostics/metrics.txt 2>&1

   echo "=== Logs ===" > diagnostics/logs.txt
   journalctl -u tado-exporter -n 100 >> diagnostics/logs.txt

   # ZIP without sensitive data
   zip -r diagnostics.zip diagnostics/
   ```

2. **Open GitHub Issue:**
   - Include diagnostic bundle
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
- [README.md](README.md) - Quick start
