package transcoder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

type ffmpegBackend struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

var ffmpegParamRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

func newFFmpegBackend(ctx context.Context, cfg config.TranscodeConfig, upstream string, log *logger.Logger) (Backend, error) {
	vCodec := "libx264"
	if cfg.VideoCodec != "" {
		vCodec = cfg.VideoCodec
	}
	if !ffmpegParamRegex.MatchString(vCodec) {
		return nil, fmt.Errorf("invalid video codec: %s", vCodec)
	}
	aCodec := "aac"
	if cfg.AudioCodec != "" {
		aCodec = cfg.AudioCodec
	}
	if !ffmpegParamRegex.MatchString(aCodec) {
		return nil, fmt.Errorf("invalid audio codec: %s", aCodec)
	}

	if cfg.Preset != "" {
		if !ffmpegParamRegex.MatchString(cfg.Preset) {
			return nil, fmt.Errorf("invalid preset: %s", cfg.Preset)
		}
	}

	var gopFlags []string
	var gopErr error
	if cfg.GOP != "" {
		gopFlags, gopErr = gopArgs(cfg.GOP)
		if gopErr != nil {
			return nil, gopErr
		}
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg binary not found: %w", err)
	}

	args := []string{
		"-re",
		"-i", "pipe:0",
		"-c:v", vCodec,
		"-c:a", aCodec,
	}

	if cfg.Preset != "" {
		args = append(args, "-preset", cfg.Preset)
	}
	if cfg.CRF > 0 {
		args = append(args, "-crf", fmt.Sprintf("%d", cfg.CRF))
	}
	args = append(args, gopFlags...)

	args = append(args, "-f", "flv", upstream)

	log.Info("starting ffmpeg", "args", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	return &ffmpegBackend{
		cmd:   cmd,
		stdin: stdin,
	}, nil
}

func (t *ffmpegBackend) Write(p []byte) (int, error) {
	return t.stdin.Write(p)
}

func (t *ffmpegBackend) Close() error {
	_ = t.stdin.Close()
	return t.cmd.Wait()
}
