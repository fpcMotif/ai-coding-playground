package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.ListenAddr != ":1935" {
		t.Fatalf("listen addr = %s", cfg.ListenAddr)
	}
	if time.Duration(cfg.IdleTimeout) != 30*time.Second {
		t.Fatalf("idle timeout = %v", time.Duration(cfg.IdleTimeout))
	}
	if cfg.ReadBuffer != 64*1024 || cfg.WriteBuffer != 64*1024 {
		t.Fatalf("buffer sizes = %d/%d", cfg.ReadBuffer, cfg.WriteBuffer)
	}
}

func TestLoadFileAndValidate(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	data := []byte(`{"listen_addr":":1935","upstream":"rtmp://example/app/stream","idle_timeout":"15s","read_buffer":4096,"write_buffer":4096}`)
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
}

func TestValidateMissingFields(t *testing.T) {
	cfg := Default()
	cfg.Upstream = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateTranscodeGOP(t *testing.T) {
	cfg := Default()
	cfg.Upstream = "rtmp://example.com/app/stream"
	cfg.Transcode.Enabled = true

	cfg.Transcode.GOP = "2s"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected duration gop to be valid, got %v", err)
	}

	cfg.Transcode.GOP = "60"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected frame gop to be valid, got %v", err)
	}

	cfg.Transcode.GOP = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid gop to fail validation")
	}
}

func TestValidateTLSConfig(t *testing.T) {
	cfg := Default()
	cfg.Upstream = "rtmp://example.com/app/stream"
	cfg.Security.TLSEnabled = true

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected tls validation error without cert/key")
	}

	cfg.Security.TLSCert = "cert.pem"
	cfg.Security.TLSKey = "key.pem"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected tls config to validate, got %v", err)
	}
}

func TestValidateUpstreamsList(t *testing.T) {
	cfg := Default()
	cfg.Upstream = ""
	cfg.Upstreams = []UpstreamEndpoint{
		{URL: "rtmp://example.com/app/stream"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected upstream list to validate, got %v", err)
	}

	cfg.Upstreams = []UpstreamEndpoint{
		{URL: ""},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected empty upstream url to fail validation")
	}

	cfg.Upstreams = []UpstreamEndpoint{
		{URL: "rtmp://example.com/app/stream", Weight: -1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected negative weight to fail validation")
	}
}

func TestValidateUpstreamStrategy(t *testing.T) {
	cfg := Default()
	cfg.Upstream = "rtmp://example.com/app/stream"
	cfg.UpstreamStrategy = "not-a-strategy"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid upstream_strategy to fail validation")
	}
}
