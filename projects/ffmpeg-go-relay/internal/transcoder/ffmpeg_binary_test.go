package transcoder

import (
	"context"
	"strings"
	"testing"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

func TestNewFFmpegBackend_ValidParams(t *testing.T) {
	// Only run if ffmpeg is in PATH, otherwise it's skipped implicitly
	// Actually we should test that the validation passes, but since exec.LookPath
	// might fail if ffmpeg is not installed, we can just check if the error is
	// specifically about the validation or not.

	cfg := config.TranscodeConfig{
		VideoCodec: "libx264",
		AudioCodec: "aac",
		Preset:     "ultrafast",
	}

	log := logger.New()

	_, err := newFFmpegBackend(context.Background(), cfg, "rtmp://localhost/app", log)
	if err != nil && strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected valid parameters to pass validation, got: %v", err)
	}
}

func TestNewFFmpegBackend_InvalidVideoCodec(t *testing.T) {
	cfg := config.TranscodeConfig{
		VideoCodec: "-vcodec foo",
	}

	log := logger.New()

	_, err := newFFmpegBackend(context.Background(), cfg, "rtmp://localhost/app", log)
	if err == nil {
		t.Fatal("expected error for invalid video codec, got nil")
	}
	if !strings.Contains(err.Error(), "invalid video codec") {
		t.Fatalf("expected error about invalid video codec, got: %v", err)
	}
}

func TestNewFFmpegBackend_InvalidAudioCodec(t *testing.T) {
	cfg := config.TranscodeConfig{
		AudioCodec: "aac -acodec bar",
	}

	log := logger.New()

	_, err := newFFmpegBackend(context.Background(), cfg, "rtmp://localhost/app", log)
	if err == nil {
		t.Fatal("expected error for invalid audio codec, got nil")
	}
	if !strings.Contains(err.Error(), "invalid audio codec") {
		t.Fatalf("expected error about invalid audio codec, got: %v", err)
	}
}

func TestNewFFmpegBackend_InvalidPreset(t *testing.T) {
	cfg := config.TranscodeConfig{
		Preset: "ultrafast; rm -rf /",
	}

	log := logger.New()

	_, err := newFFmpegBackend(context.Background(), cfg, "rtmp://localhost/app", log)
	if err == nil {
		t.Fatal("expected error for invalid preset, got nil")
	}
	if !strings.Contains(err.Error(), "invalid preset") {
		t.Fatalf("expected error about invalid preset, got: %v", err)
	}
}

func TestNewFFmpegBackend_InvalidGOP(t *testing.T) {
	cfg := config.TranscodeConfig{
		GOP: "2s; rm -rf /",
	}

	log := logger.New()

	_, err := newFFmpegBackend(context.Background(), cfg, "rtmp://localhost/app", log)
	if err == nil {
		t.Fatal("expected error for invalid gop, got nil")
	}
	if !strings.Contains(err.Error(), "invalid gop") {
		t.Fatalf("expected error about invalid gop, got: %v", err)
	}
}
