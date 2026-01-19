package transcoder

import (
	"testing"

	"ffmpeg-go-relay/internal/config"
)

func TestResolveBackendDefault(t *testing.T) {
	backend, err := resolveBackend(config.TranscodeConfig{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if backend != backendFFmpeg {
		t.Fatalf("expected %s, got %s", backendFFmpeg, backend)
	}
}

func TestResolveBackendExplicit(t *testing.T) {
	backend, err := resolveBackend(config.TranscodeConfig{Backend: "libav"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if backend != backendLibAV {
		t.Fatalf("expected %s, got %s", backendLibAV, backend)
	}
}

func TestResolveBackendUnknown(t *testing.T) {
	if _, err := resolveBackend(config.TranscodeConfig{Backend: "unknown"}); err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
