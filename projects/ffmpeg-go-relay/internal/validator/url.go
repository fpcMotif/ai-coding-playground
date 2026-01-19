package validator

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateUpstreamURL validates an upstream RTMP URL to prevent SSRF attacks.
// It checks:
// - URL format and scheme
// - Host is not in private/reserved IP ranges
// - Port is within valid range
func ValidateUpstreamURL(upstream string) error {
	if upstream == "" {
		return fmt.Errorf("upstream URL cannot be empty")
	}

	// Parse URL - prepend scheme if missing
	if !strings.Contains(upstream, "://") {
		upstream = "rtmp://" + upstream
	}

	parsed, err := url.Parse(upstream)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}

	// Validate scheme
	if parsed.Scheme != "rtmp" && parsed.Scheme != "rtmps" && parsed.Scheme != "rtsps" && parsed.Scheme != "rtsp" {
		return fmt.Errorf("unsupported scheme %q (must be rtmp, rtmps, rtsp, or rtsps)", parsed.Scheme)
	}

	// Extract host and port
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("upstream URL must include a host")
	}

	// Validate port if specified
	portStr := parsed.Port()
	if portStr != "" {
		port := 0
		_, err := fmt.Sscanf(portStr, "%d", &port)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %q: must be 1-65535", portStr)
		}
	}

	// Reject private/reserved IP ranges (SSRF prevention)
	if err := isReservedIP(host); err != nil {
		return err
	}

	return nil
}

// isReservedIP checks if the host is in a private, loopback, or cloud metadata IP range
func isReservedIP(host string) error {
	// Try to parse as IP address
	ip := net.ParseIP(host)
	if ip == nil {
		// If it's a hostname, do basic DNS validation
		// In production, you might want to resolve and check the IP too
		return validateHostname(host)
	}

	// Check for loopback (127.0.0.0/8)
	if ip.IsLoopback() {
		return fmt.Errorf("upstream cannot be loopback address %s", host)
	}

	// Check for private ranges (RFC 1918)
	if ip.IsPrivate() {
		return fmt.Errorf("upstream cannot be in private IP range: %s", host)
	}

	// Check for link-local (169.254.0.0/16)
	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("upstream cannot be link-local address: %s", host)
	}

	// Check for multicast (224.0.0.0/4)
	if ip.IsMulticast() {
		return fmt.Errorf("upstream cannot be multicast address: %s", host)
	}

	// Check for unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("upstream cannot be unspecified address: %s", host)
	}

	// Check for cloud metadata endpoints
	if ip.String() == "169.254.169.254" {
		return fmt.Errorf("upstream cannot be cloud metadata endpoint: %s", host)
	}

	return nil
}

// validateHostname does basic validation on hostnames
// In production, consider adding DNS validation or resolution
func validateHostname(host string) error {
	// Reject common cloud metadata endpoints by hostname
	blocked := []string{
		"localhost",
		"169.254.169.254", // AWS metadata endpoint
		"metadata.google.internal",
		"kubernetes.default",
		"host.docker.internal",
	}

	for _, blocked := range blocked {
		if host == blocked {
			return fmt.Errorf("upstream cannot use %s", host)
		}
	}

	// Basic hostname validation - should contain at least one dot
	// or be localhost (which we already blocked)
	// Allow any hostname that doesn't match blocked list
	return nil
}
