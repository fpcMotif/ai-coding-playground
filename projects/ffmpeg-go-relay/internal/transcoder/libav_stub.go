//go:build !libav

package transcoder

import (
	"context"
	"fmt"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

func newLibAVBackend(ctx context.Context, cfg config.TranscodeConfig, upstream string, log *logger.Logger) (Backend, error) {
	return nil, fmt.Errorf("libav backend not enabled; build with -tags libav")
}
