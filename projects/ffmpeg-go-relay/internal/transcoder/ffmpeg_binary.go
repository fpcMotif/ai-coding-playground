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

var validParamRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

type ffmpegBackend struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func newFFmpegBackend(ctx context.Context, cfg config.TranscodeConfig, upstream string, log *logger.Logger) (Backend, error) {

	vCodec := "libx264"
	if cfg.VideoCodec != "" {
		if !validParamRegex.MatchString(cfg.VideoCodec) {
			return nil, fmt.Errorf("invalid video codec: %s", cfg.VideoCodec)
		}
		vCodec = cfg.VideoCodec
	}
	aCodec := "aac"
	if cfg.AudioCodec != "" {
		if !validParamRegex.MatchString(cfg.AudioCodec) {
			return nil, fmt.Errorf("invalid audio codec: %s", cfg.AudioCodec)
		}
		aCodec = cfg.AudioCodec
	}

	args := []string{
		"-re",
		"-i", "pipe:0",
		"-c:v", vCodec,
		"-c:a", aCodec,
	}

	if cfg.Preset != "" {
		if !validParamRegex.MatchString(cfg.Preset) {
			return nil, fmt.Errorf("invalid preset: %s", cfg.Preset)
		}
		args = append(args, "-preset", cfg.Preset)
	}
	if cfg.CRF > 0 {
		args = append(args, "-crf", fmt.Sprintf("%d", cfg.CRF))
	}
	if cfg.GOP != "" {
		if !validParamRegex.MatchString(cfg.GOP) {
			return nil, fmt.Errorf("invalid gop: %s", cfg.GOP)
		}
		gopFlags, err := gopArgs(cfg.GOP)
		if err != nil {
			return nil, err
		}
		args = append(args, gopFlags...)
	}

	args = append(args, "-f", "flv", upstream)

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg binary not found: %w", err)
	}

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
