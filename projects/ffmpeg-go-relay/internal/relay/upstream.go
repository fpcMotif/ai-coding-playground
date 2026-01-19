package relay

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const (
	defaultRTMPPort = "1935"
	defaultRTSPPort = "554"
)

// UpstreamInfo describes how to dial an upstream endpoint.
type UpstreamInfo struct {
	Raw     string
	Scheme  string
	Host    string
	Port    string
	Address string
	UseTLS  bool
}

// ParseUpstream normalizes an upstream string and returns connection info.
func ParseUpstream(raw string) (UpstreamInfo, error) {
	if raw == "" {
		return UpstreamInfo{}, fmt.Errorf("upstream is empty")
	}

	normalized := raw
	if !strings.Contains(raw, "://") {
		normalized = "rtmp://" + raw
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return UpstreamInfo{}, fmt.Errorf("parse upstream: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "rtmp", "rtmps", "rtsp", "rtsps":
	default:
		return UpstreamInfo{}, fmt.Errorf("unsupported upstream scheme %q", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return UpstreamInfo{}, fmt.Errorf("upstream host is empty")
	}

	port := parsed.Port()
	if port == "" {
		port = defaultPortForScheme(scheme)
	}

	address := net.JoinHostPort(host, port)

	return UpstreamInfo{
		Raw:     raw,
		Scheme:  scheme,
		Host:    host,
		Port:    port,
		Address: address,
		UseTLS:  scheme == "rtmps" || scheme == "rtsps",
	}, nil
}

func defaultPortForScheme(scheme string) string {
	switch scheme {
	case "rtsp", "rtsps":
		return defaultRTSPPort
	default:
		return defaultRTMPPort
	}
}
