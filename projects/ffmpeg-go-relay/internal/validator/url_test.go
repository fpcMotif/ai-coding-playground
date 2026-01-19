package validator

import (
	"testing"
)

func TestValidateUpstreamURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Valid URLs
		{
			name:    "valid RTMP URL with host",
			url:     "rtmp://example.com/app/stream",
			wantErr: false,
		},
		{
			name:    "valid RTMP URL with port",
			url:     "rtmp://example.com:1935/app/stream",
			wantErr: false,
		},
		{
			name:    "valid RTMPS URL",
			url:     "rtmps://example.com/app/stream",
			wantErr: false,
		},
		{
			name:    "valid RTSP URL",
			url:     "rtsp://example.com/stream",
			wantErr: false,
		},
		{
			name:    "valid RTSPS URL",
			url:     "rtsps://example.com/stream",
			wantErr: false,
		},
		{
			name:    "public IPv4 address",
			url:     "rtmp://8.8.8.8:1935/app",
			wantErr: false,
		},

		// Invalid schemes
		{
			name:    "invalid scheme http",
			url:     "http://example.com/app",
			wantErr: true,
		},
		{
			name:    "invalid scheme https",
			url:     "https://example.com/app",
			wantErr: true,
		},
		{
			name:    "invalid scheme file",
			url:     "file:///etc/passwd",
			wantErr: true,
		},

		// Private IP ranges (RFC 1918)
		{
			name:    "private IP 192.168.x.x",
			url:     "rtmp://192.168.1.1/app",
			wantErr: true,
		},
		{
			name:    "private IP 10.x.x.x",
			url:     "rtmp://10.0.0.1/app",
			wantErr: true,
		},
		{
			name:    "private IP 172.16.x.x",
			url:     "rtmp://172.16.0.1/app",
			wantErr: true,
		},

		// Loopback
		{
			name:    "loopback 127.0.0.1",
			url:     "rtmp://127.0.0.1/app",
			wantErr: true,
		},
		{
			name:    "loopback localhost",
			url:     "rtmp://localhost/app",
			wantErr: true,
		},

		// Link-local
		{
			name:    "link-local 169.254.x.x",
			url:     "rtmp://169.254.1.1/app",
			wantErr: true,
		},

		// Cloud metadata endpoints
		{
			name:    "AWS metadata endpoint",
			url:     "rtmp://169.254.169.254/app",
			wantErr: true,
		},
		{
			name:    "Google Cloud metadata",
			url:     "rtmp://metadata.google.internal/app",
			wantErr: true,
		},
		{
			name:    "Kubernetes default service",
			url:     "rtmp://kubernetes.default/app",
			wantErr: true,
		},
		{
			name:    "Docker host endpoint",
			url:     "rtmp://host.docker.internal/app",
			wantErr: true,
		},

		// Multicast
		{
			name:    "multicast address",
			url:     "rtmp://224.0.0.1/app",
			wantErr: true,
		},

		// Unspecified
		{
			name:    "unspecified 0.0.0.0",
			url:     "rtmp://0.0.0.0/app",
			wantErr: true,
		},

		// Invalid ports
		{
			name:    "port 0",
			url:     "rtmp://example.com:0/app",
			wantErr: true,
		},
		{
			name:    "port 65536 (out of range)",
			url:     "rtmp://example.com:65536/app",
			wantErr: true,
		},
		{
			name:    "invalid port string",
			url:     "rtmp://example.com:abc/app",
			wantErr: true,
		},

		// Missing host
		{
			name:    "missing host",
			url:     "rtmp:///app",
			wantErr: true,
		},

		// Empty URL
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},

		// URL without scheme (should add rtmp://)
		{
			name:    "URL without scheme (public IP)",
			url:     "example.com:1935",
			wantErr: false,
		},
		{
			name:    "URL without scheme (private IP)",
			url:     "192.168.1.1:1935",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpstreamURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpstreamURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsReservedIP(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		// Reserved
		{"loopback 127.0.0.1", "127.0.0.1", true},
		{"loopback 127.0.0.2", "127.0.0.2", true},
		{"private 192.168.1.1", "192.168.1.1", true},
		{"private 10.0.0.1", "10.0.0.1", true},
		{"private 172.16.0.1", "172.16.0.1", true},
		{"link-local 169.254.1.1", "169.254.1.1", true},
		{"multicast 224.0.0.1", "224.0.0.1", true},
		{"unspecified 0.0.0.0", "0.0.0.0", true},
		{"metadata 169.254.169.254", "169.254.169.254", true},

		// Public
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 1.1.1.1", "1.1.1.1", false},
		{"public 208.67.222.222", "208.67.222.222", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isReservedIP(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("isReservedIP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"valid hostname", "example.com", false},
		{"valid subdomain", "relay.example.com", false},
		{"localhost blocked", "localhost", true},
		{"metadata.google.internal blocked", "metadata.google.internal", true},
		{"kubernetes.default blocked", "kubernetes.default", true},
		{"host.docker.internal blocked", "host.docker.internal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostname(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHostname() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
