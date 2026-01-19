package transcoder

import (
	"context"
	"fmt"
	"io"
	"strings"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

const (
	backendFFmpeg = "ffmpeg"
	backendLibAV  = "libav"
)

type Backend interface {
	io.WriteCloser
}

func New(ctx context.Context, cfg config.TranscodeConfig, upstream string, log *logger.Logger) (Backend, error) {
	backend, err := resolveBackend(cfg)
	if err != nil {
		return nil, err
	}

	switch backend {
	case backendFFmpeg:
		return newFFmpegBackend(ctx, cfg, upstream, log)
	case backendLibAV:
		return newLibAVBackend(ctx, cfg, upstream, log)
	default:
		return nil, fmt.Errorf("unknown transcode backend: %s", backend)
	}
}

func resolveBackend(cfg config.TranscodeConfig) (string, error) {
	backend := strings.TrimSpace(strings.ToLower(cfg.Backend))
	if backend == "" {
		return backendFFmpeg, nil
	}
	if backend != backendFFmpeg && backend != backendLibAV {
		return "", fmt.Errorf("unknown transcode backend: %s", cfg.Backend)
	}
	return backend, nil
}
