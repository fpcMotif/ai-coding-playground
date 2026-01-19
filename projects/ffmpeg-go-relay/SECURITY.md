# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in FFmpeg-Go-Relay, please report it to [security@example.com](mailto:security@example.com) instead of using the public issue tracker.

### Responsible Disclosure

Please provide:
1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if available)

We will acknowledge receipt within 48 hours and provide a status update within 7 days.

## Security Measures

### Authentication

#### Token-Based Authentication
- Tokens are validated on every client connection
- No token means connection is rejected if auth is enabled
- Tokens are stored in memory (not persisted)
- Tokens should be rotated regularly

#### Best Practices
- Use strong, random tokens (minimum 32 characters)
- Rotate tokens every 30-90 days
- Store tokens in a secrets management system (Vault, AWS Secrets Manager, etc.)
- Never commit tokens to version control

### Network Security

#### SSRF Prevention
- Upstream URLs are validated to prevent Server-Side Request Forgery
- Private IP ranges are blocked (RFC 1918, 127.0.0.0/8, 169.254.0.0/16)
- Cloud metadata endpoints are blocked (169.254.169.254, etc.)
- Only RTMP/RTMPS schemes are allowed

#### Blocked IP Ranges
- 0.0.0.0/8 (This network)
- 10.0.0.0/8 (Private)
- 127.0.0.0/8 (Loopback)
- 169.254.0.0/16 (Link-local)
- 172.16.0.0/12 (Private)
- 192.168.0.0/16 (Private)
- 224.0.0.0/4 (Multicast)

#### Blocked Hostnames
- localhost
- metadata.google.internal
- kubernetes.default
- host.docker.internal

### TLS/SSL

#### Configuration
```json
{
  "security": {
    "tls_enabled": true,
    "tls_cert": "/path/to/cert.pem",
    "tls_key": "/path/to/key.pem"
  }
}
```

#### Certificate Requirements
- Use valid, signed certificates (not self-signed in production)
- Use certificates with proper key usage extensions
- Ensure certificates are not expired
- Use strong key sizes (RSA 2048+ or ECDSA P-256+)

#### TLS Versions
- Minimum: TLS 1.2
- Recommended: TLS 1.3
- Disable older versions in firewall/load balancer

### Rate Limiting

#### Configuration
```json
{
  "rate_limit": {
    "enabled": true,
    "requests_per_sec": 100.0,
    "burst": 10
  }
}
```

#### Purpose
- Prevent brute force attacks
- Protect against DoS attempts
- Fair resource allocation

#### Best Practices
- Set `requests_per_sec` based on expected legitimate load
- Monitor rate limit rejections for anomalies
- Whitelist trusted IPs if needed
- Use with connection limiting

### Connection Limiting

#### Configuration
```json
{
  "connection_limit": {
    "max_total_connections": 1000,
    "max_per_ip": 10
  }
}
```

#### Purpose
- Prevent resource exhaustion
- Limit per-IP connections to prevent single-source DoS
- Fair resource distribution

#### Best Practices
- Set limits based on available resources
- Monitor connection metrics
- Alert on limit threshold violations
- Consider proxy/load balancer topology

### Logging and Monitoring

#### Log Contents
- Connection events (allowed/rejected)
- Authentication failures
- Rate limit violations
- Upstream connection issues
- Error conditions

#### Log Security
- Logs contain IP addresses and connection metadata
- Sanitize logs before sharing in bug reports
- Store logs securely
- Implement log rotation and retention

#### Monitoring
- Monitor rate limit rejections for unusual patterns
- Alert on authentication failures
- Track connection limits and GC metrics
- Use structured logging for easy parsing

### Code Security

#### Static Analysis
- Run `gosec` in CI/CD pipeline
- Address high-severity findings immediately
- Review and document low-severity findings

#### Dependencies
- Keep Go updated to latest patch version
- Run `go mod tidy` regularly
- Monitor for vulnerable dependencies
- Use `go mod audit` or similar tools

#### Input Validation
- Upstream URL validation (SSRF prevention)
- Buffer size limits (4KB-1MB)
- Configuration validation on startup
- Duration parsing with error handling

### Docker Security

#### Image Security
- Multi-stage build to minimize image size
- Run as non-root user (UID 1000)
- Use minimal base image (alpine)
- Scan image for vulnerabilities

#### Container Runtime
- Use security context to prevent privilege escalation
- Mount volumes as read-only when possible
- Limit resource consumption (CPU, memory)
- Use network policies to restrict traffic

#### Example Kubernetes Security Context
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
```

### Operational Security

#### Deployment
- Use infrastructure-as-code for consistency
- Implement secrets management
- Use configuration management
- Version control everything except secrets

#### Access Control
- Restrict SSH/RDP access to deployment servers
- Use bastion hosts for administrative access
- Implement strong authentication (2FA/MFA)
- Audit access logs

#### Incident Response
- Document incident response procedures
- Have backup relay servers ready
- Test failover procedures regularly
- Keep security contacts up-to-date

## Security Checklist

### Pre-Deployment

- [ ] Authentication enabled with strong tokens
- [ ] TLS enabled with valid certificates
- [ ] Rate limiting configured appropriately
- [ ] Connection limits set based on capacity
- [ ] Circuit breaker enabled
- [ ] Logs being collected and monitored
- [ ] Upstream URLs validated
- [ ] Firewall rules configured
- [ ] Regular backup procedure tested
- [ ] Incident response plan documented

### Post-Deployment

- [ ] Monitor authentication failure rate
- [ ] Monitor rate limit violations
- [ ] Review logs for suspicious patterns
- [ ] Check certificate expiration dates (alert 30 days before)
- [ ] Test failover procedures monthly
- [ ] Rotate authentication tokens quarterly
- [ ] Review and update security policy annually
- [ ] Run security scans on code regularly
- [ ] Monitor for vulnerable dependencies
- [ ] Keep Go runtime updated

## Common Vulnerabilities and Mitigations

### SSRF Attacks
**Risk**: Upstream URL parameter used to attack internal services
**Mitigation**: Upstream URL validation blocks private IPs and cloud metadata endpoints

### DoS Attacks
**Risk**: Attacker sends many connections/requests
**Mitigations**: Rate limiting, connection limiting, circuit breaker

### Credential Theft
**Risk**: Tokens exposed in logs or transmitted insecurely
**Mitigations**: TLS encryption, secure token storage, log sanitization

### Resource Exhaustion
**Risk**: Attacker causes memory/CPU exhaustion
**Mitigations**: Buffer pooling, connection limits, timeout settings, resource limits in containers

### Man-in-the-Middle (MitM)
**Risk**: Traffic intercepted and modified
**Mitigations**: TLS encryption, certificate validation

## Performance vs Security Trade-offs

| Setting | Security Impact | Performance Impact | Default |
|---------|-----------------|-------------------|---------|
| TLS Enabled | High | Low (-5%) | off |
| Auth Enabled | High | Negligible | off |
| Rate Limiting | Medium | Low | enabled |
| Circuit Breaker | Medium | Negligible | enabled |
| Connection Limits | Medium | Low | enabled |
| Buffer Pooling | Low | High (GC -50%) | enabled |

## Security Testing

### Automated Tests
```bash
# Run gosec
go get github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Run tests with race detector
go test -race ./...

# Check dependencies
go list -json -m all | nancy sleuth
```

### Manual Testing
1. Test authentication with invalid tokens
2. Test rate limiting with load testing tool
3. Test connection limiting behavior
4. Test SSRF validation with private IPs
5. Verify TLS certificate validation
6. Test graceful error handling

## Compliance

### Standards Covered
- OWASP Top 10 prevention
- RFC 3986 (URI Generic Syntax)
- RFC 1918 (Private Internet Addresses)
- TLS Best Practices

### Audit Recommendations
- Annual security code review
- Quarterly vulnerability scanning
- Monthly log review for anomalies
- Daily monitoring of metrics

## References

- [OWASP Web Application Security](https://owasp.org/)
- [Go Security Guidelines](https://golang.org/doc/fuzz)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Kubernetes Security](https://kubernetes.io/docs/concepts/security/)
- [TLS Best Practices](https://wiki.mozilla.org/Security/Server_Side_TLS)
