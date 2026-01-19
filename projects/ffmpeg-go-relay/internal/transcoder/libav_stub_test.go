//go:build !libav

package transcoder

import (
	"context"
	"testing"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

func TestLibAVBackendUnavailable(t *testing.T) {
	_, err := newLibAVBackend(context.Background(), config.TranscodeConfig{Backend: "libav"}, "rtmp://example.com/live", logger.New())
	if err == nil {
		t.Fatal("expected error when libav backend is unavailable")
	}
}
