package transcoder

import (
	"context"
	"strings"
	"testing"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
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

func TestFFmpegBackendValidation(t *testing.T) {
	ctx := context.Background()
	log := logger.New()
	upstream := "rtmp://localhost/app/stream"

	tests := []struct {
		name    string
		cfg     config.TranscodeConfig
		wantErr string
	}{
		{
			name: "valid params",
			cfg: config.TranscodeConfig{
				Backend:    "ffmpeg",
				VideoCodec: "libx264",
				AudioCodec: "aac",
				Preset:     "veryfast",
			},
			wantErr: "",
		},
		{
			name: "invalid video codec",
			cfg: config.TranscodeConfig{
				Backend:    "ffmpeg",
				VideoCodec: "-vcodec malicious",
			},
			wantErr: "invalid video codec: -vcodec malicious",
		},
		{
			name: "invalid audio codec",
			cfg: config.TranscodeConfig{
				Backend:    "ffmpeg",
				AudioCodec: "aac -acodec malicious",
			},
			wantErr: "invalid audio codec: aac -acodec malicious",
		},
		{
			name: "invalid preset",
			cfg: config.TranscodeConfig{
				Backend: "ffmpeg",
				Preset:  "ultrafast -f something",
			},
			wantErr: "invalid preset: ultrafast -f something",
		},
		{
			name: "invalid gop",
			cfg: config.TranscodeConfig{
				Backend: "ffmpeg",
				GOP:     "-gop malicious",
			},
			wantErr: "gop must be a positive frame count or duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newFFmpegBackend(ctx, tt.cfg, upstream, log)
			if tt.wantErr == "" {
				// We expect either no error or "ffmpeg binary not found" depending on the environment,
				// or an error regarding exec.Command because it's a mock.
				// The key is that we don't expect a validation error.
				if err != nil && strings.Contains(err.Error(), "invalid") {
					t.Errorf("unexpected validation error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}
