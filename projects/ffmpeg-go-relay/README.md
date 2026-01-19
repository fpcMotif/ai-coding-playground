# FFmpeg-Go-Relay: High-Performance RTMP Relay Server

A production-ready RTMP/FLV stream relay server written in Go, designed for low-latency streaming with built-in security, monitoring, and resilience features.

## Features

### Core Capabilities
- **Low-Latency TCP Relay**: Bidirectional TCP stream relay optimized for real-time RTMP/FLV streaming
- **RTMP/RTMPS Support**: Relay RTMP, RTMPS, RTSP, and RTSPS streams
- **Multiple Upstream Servers**: Route to different upstream servers based on configuration

### Security
- **Token-Based Authentication**: Validate clients with bearer tokens
- **TLS Support**: Encrypt connections with configurable certificates
- **SSRF Prevention**: Upstream URL validation blocks private/reserved IPs
- **Rate Limiting**: Per-IP rate limiting with configurable burst
- **Connection Limiting**: Global and per-IP connection limits
- **Non-Root User**: Docker container runs as non-root for security

### Monitoring & Observability
- **Prometheus Metrics**: Complete metrics export for monitoring
  - Active connections (gauge)
  - Total connections by status (counter)
  - Bytes transferred (counter)
  - Connection duration (histogram)
  - Latency metrics
  - Upstream error tracking
- **Health Endpoints**: `/health`, `/ready`, `/livez`, `/status` for load balancer integration
- **Structured Logging**: JSON logs with connection tracking for debugging

### Resilience & Performance
- **Circuit Breaker**: Automatically handle upstream failures
- **Exponential Backoff Retry**: Intelligent retry logic with jitter
- **Buffer Pooling**: Reduce GC pressure with sync.Pool-based buffer reuse
- **Connection Pooling**: Reuse upstream connections efficiently
- **Graceful Shutdown**: Clean connection draining with timeout
- **Dynamic Deadlines**: Prevent false idle timeouts during streaming

### Configuration
- **JSON Configuration**: Easy deployment configuration
- **String Duration Format**: Human-readable timeouts ("30s", "5m")
- **Validation**: Comprehensive config validation on startup
- **Environment Variables**: Override any config via env vars

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone repository
git clone https://github.com/your-org/ffmpeg-go-relay.git
cd ffmpeg-go-relay

# Start stack with relay, Prometheus, and Grafana
docker-compose up -d

# Access services:
# - Relay metrics: http://localhost:8080/metrics
# - Relay health: http://localhost:8080/health
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
```

### Using Docker

```bash
# Build image
docker build -t ffmpeg-go-relay .

# Run with default config
docker run -p 1935:1935 -p 8080:8080 \
  -e UPSTREAM=rtmp://upstream.example.com:1935/app/stream \
  ffmpeg-go-relay

# Run with custom config
docker run -p 1935:1935 -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json:ro \
  ffmpeg-go-relay
```

### Building from Source

```bash
# Requirements: Go 1.21+

# Clone and build
git clone https://github.com/your-org/ffmpeg-go-relay.git
cd ffmpeg-go-relay
go build -o relay ./cmd/relay

# Run with config
./relay config.json

# Or use environment variables
LISTEN_ADDR=:1935 UPSTREAM=rtmp://upstream.example.com:1935/app/stream ./relay
```

## Configuration

### Basic Configuration

```json
{
  "listen_addr": ":1935",
  "http_addr": ":8080",
  "upstream": "rtmp://upstream.example.com:1935/app/stream",
  "idle_timeout": "30s",
  "read_buffer": 65536,
  "write_buffer": 65536
}
```

### Full Configuration with All Features

See `config.example.json` for a complete example with:
- Authentication tokens
- TLS settings
- Rate limiting
- Connection limits
- Circuit breaker settings
- Retry policy

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `listen_addr` | string | `:1935` | Listen address for RTMP clients |
| `http_addr` | string | `:8080` | HTTP address for health and metrics (empty to disable) |
| `upstream` | string | required | Upstream RTMP server (rtmp://host:port/path) |
| `idle_timeout` | duration | `30s` | Connection idle timeout |
| `read_buffer` | int | `65536` | TCP read buffer size (4KB-1MB) |
| `write_buffer` | int | `65536` | TCP write buffer size (4KB-1MB) |

### Security Configuration

```json
{
  "security": {
    "auth_enabled": true,
    "auth_tokens": ["token-1", "token-2"],
    "tls_enabled": false,
    "tls_cert": "/path/to/cert.pem",
    "tls_key": "/path/to/key.pem"
  }
}
```

### Rate Limiting

```json
{
  "rate_limit": {
    "enabled": true,
    "requests_per_sec": 100.0,
    "burst": 10
  }
}
```

### Connection Limiting

```json
{
  "connection_limit": {
    "max_total_connections": 1000,
    "max_per_ip": 10
  }
}
```

### Circuit Breaker

```json
{
  "circuit_breaker": {
    "enabled": true,
    "max_failures": 5,
    "reset_timeout_sec": 30,
    "success_threshold": 2
  }
}
```

### Retry Policy

```json
{
  "retry": {
    "enabled": true,
    "max_attempts": 3,
    "initial_delay_sec": 1,
    "max_delay_sec": 30,
    "multiplier": 2.0,
    "jitter_fraction": 0.1
  }
}
```

## Monitoring

### Prometheus Metrics

The relay exports metrics at `http://localhost:8080/metrics`:

```
# Active connections
rtmp_relay_active_connections

# Connection counters
rtmp_relay_connections_total{status="success|error|rejected"}

# Bytes transferred
rtmp_relay_bytes_total{direction="upstream|downstream"}

# Connection duration histogram
rtmp_relay_connection_duration_seconds_bucket

# Error tracking
rtmp_relay_upstream_errors_total{error_type="..."}

# Rate limit rejections
rtmp_relay_rate_limit_rejections_total

# Auth failures
rtmp_relay_auth_failures_total
```

### Health Endpoints

- **GET /** - Returns basic service info
- **GET /health** - Returns 200 if running
- **GET /ready** - Returns 200 if upstream is reachable
- **GET /livez** - Returns 200 (always alive)
- **GET /status** - Returns detailed connection and rate limit stats
- **GET /metrics** - Prometheus metrics

### Grafana Dashboard

The docker-compose includes pre-configured Prometheus and Grafana:

```bash
# Access Grafana
open http://localhost:3000

# Default credentials: admin/admin

# Dashboard shows:
# - Active connections over time
# - Bytes transferred per second
# - Latency percentiles (p50, p99, p99.9)
# - Error rates by type
# - Rate limit rejections
# - Connection limit rejections
```

## Performance Tuning

### Buffer Sizes

Adjust based on network conditions and stream bitrate:

```json
{
  "read_buffer": 131072,   // 128KB for high-throughput
  "write_buffer": 131072
}
```

### Connection Pooling

Optimize upstream connection reuse:

```json
{
  "retry": {
    "max_attempts": 5,
    "initial_delay_sec": 1,
    "multiplier": 1.5
  }
}
```

### Rate Limiting

Calculate based on expected load:

```
requests_per_sec = (max_bitrate_mbps * 1_000_000 / 8) / avg_request_size
burst = peak_requests - requests_per_sec * 1
```

## Deployment

### Kubernetes

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: relay
spec:
  containers:
  - name: relay
    image: ffmpeg-go-relay:latest
    ports:
    - containerPort: 1935
    - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /livez
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 30
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
```

### Docker Swarm

```bash
docker service create \
  --name relay \
  -p 1935:1935 \
  -p 8080:8080 \
  --health-cmd='wget -q -O- http://localhost:8080/health' \
  --health-interval=30s \
  ffmpeg-go-relay
```

## Testing

### Run Tests

```bash
# Unit tests
go test ./... -v

# With coverage
go test ./... -cover

# Integration tests
go test ./test -v -timeout 60s

# Benchmarks
go test ./test -bench=. -benchmem
```

### Load Testing

```bash
# Using ffmpeg to send stream
ffmpeg -f lavfi -i testsrc=s=1280x720:d=3600 -f lavfi -i sine=f=440:d=3600 \
  -pix_fmt yuv420p -c:v libx264 -c:a aac \
  -rtmp_live live -f flv rtmp://localhost:1935/app/stream

# Monitor with curl
while true; do curl -s http://localhost:8080/status | jq '.connections'; sleep 5; done
```

## Architecture

### Components

1. **Relay Server** (`internal/relay/server.go`)
   - TCP listener for RTMP clients
   - Connection handler with middleware
   - Bidirectional stream relay

2. **Security** (`internal/auth/`, `internal/validator/`)
   - Token-based authentication
   - SSRF prevention for upstream URLs
   - TLS support

3. **Middleware** (`internal/middleware/`)
   - Rate limiting per IP
   - Connection limiting (global and per-IP)
   - Request counting

4. **Resilience** (`internal/circuit/`, `internal/retry/`)
   - Circuit breaker for upstream failures
   - Exponential backoff with jitter

5. **Performance** (`internal/pool/`)
   - Buffer pooling for GC reduction
   - Connection reuse

6. **Observability** (`internal/metrics/`, `internal/httpserver/`)
   - Prometheus metrics
   - Health check endpoints
   - Structured JSON logging

## Troubleshooting

### Connection Rejected

Check if rate limiting or connection limits are exceeded:

```bash
curl http://localhost:8080/status | jq '.rate_limit, .connections'
```

### Upstream Connection Failures

Check circuit breaker status and retry logs:

```bash
tail -f relay.log | grep -i "circuit\|upstream\|retry"
```

### High Latency

- Check buffer sizes (increase `read_buffer`/`write_buffer`)
- Monitor GC pressure: `go tool pprof http://localhost:6060/debug/pprof/heap`
- Check network conditions

### Memory Usage

- Enable buffer pooling in config
- Reduce idle timeout to close stale connections
- Monitor with: `curl http://localhost:8080/status`

## Security Considerations

### Authentication

- Always enable authentication in production
- Rotate tokens regularly
- Use strong, random tokens

### TLS

- Use valid certificates (not self-signed in production)
- Enable TLS for all client connections
- Use modern TLS versions (1.2+)

### Network

- Restrict upstream URL to public IPs
- Use firewall rules to limit client IP ranges
- Monitor for rate limiting anomalies

### Running as Non-Root

The Docker image runs as user `appuser` (UID 1000) for security. Ensure:
- Config files are readable by UID 1000
- Log directories are writable by UID 1000

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

[Your License Here]

## Support

For issues, feature requests, or questions:
- [GitHub Issues](https://github.com/your-org/ffmpeg-go-relay/issues)
- [Email](mailto:support@example.com)

## Roadmap

- [ ] HTTP/3 support
- [ ] WebRTC relay
- [ ] Redis-based distributed caching
- [ ] Kubernetes operator
- [ ] Horizontal scaling with load balancing
- [ ] Plugin system for custom processing
