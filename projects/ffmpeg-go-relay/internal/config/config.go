package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ffmpeg-go-relay/internal/validator"
)

// SecurityConfig defines security settings.
type SecurityConfig struct {
	AuthEnabled bool     `json:"auth_enabled"`
	AuthTokens  []string `json:"auth_tokens"`
	TLSEnabled  bool     `json:"tls_enabled"`
	TLSCert     string   `json:"tls_cert"`
	TLSKey      string   `json:"tls_key"`
}

// RateLimitConfig defines rate limiting settings.
type RateLimitConfig struct {
	Enabled        bool    `json:"enabled"`
	RequestsPerSec float64 `json:"requests_per_sec"`
	Burst          int     `json:"burst"`
}

// ConnectionLimitConfig defines connection limit settings.
type ConnectionLimitConfig struct {
	MaxTotal int64 `json:"max_total_connections"`
	MaxPerIP int64 `json:"max_per_ip"`
}

// CircuitBreakerConfig defines circuit breaker settings.
type CircuitBreakerConfig struct {
	Enabled         bool  `json:"enabled"`
	MaxFailures     int32 `json:"max_failures"`
	ResetTimeoutSec int   `json:"reset_timeout_sec"`
	SuccessThresh   int32 `json:"success_threshold"`
}

// RetryConfig defines retry settings.
type RetryConfig struct {
	Enabled         bool    `json:"enabled"`
	MaxAttempts     int     `json:"max_attempts"`
	InitialDelaySec int     `json:"initial_delay_sec"`
	MaxDelaySec     int     `json:"max_delay_sec"`
	Multiplier      float64 `json:"multiplier"`
	JitterFraction  float64 `json:"jitter_fraction"`
}

// UpstreamEndpoint defines a single upstream target.
type UpstreamEndpoint struct {
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

// UpstreamHealthCheckConfig defines health check settings for upstreams.
type UpstreamHealthCheckConfig struct {
	Enabled     bool `json:"enabled"`
	IntervalSec int  `json:"interval_sec"`
	TimeoutSec  int  `json:"timeout_sec"`
}

// Config defines server settings.
type Config struct {
	ListenAddr          string                    `json:"listen_addr"`
	HTTPAddr            string                    `json:"http_addr"`
	Upstream            string                    `json:"upstream"`
	Upstreams           []UpstreamEndpoint        `json:"upstreams,omitempty"`
	UpstreamStrategy    string                    `json:"upstream_strategy,omitempty"`
	UpstreamHealthCheck UpstreamHealthCheckConfig `json:"upstream_health_check,omitempty"`
	IdleTimeout         Duration                  `json:"idle_timeout"`
	ReadBuffer          int                       `json:"read_buffer"`
	WriteBuffer         int                       `json:"write_buffer"`
	Security            SecurityConfig            `json:"security,omitempty"`
	RateLimit           RateLimitConfig           `json:"rate_limit,omitempty"`
	ConnectionLimit     ConnectionLimitConfig     `json:"connection_limit,omitempty"`
	CircuitBreaker      CircuitBreakerConfig      `json:"circuit_breaker,omitempty"`
	Retry               RetryConfig               `json:"retry,omitempty"`
	Transcode           TranscodeConfig           `json:"transcode,omitempty"`
}

// TranscodeConfig defines transcoding settings.
type TranscodeConfig struct {
	Enabled    bool   `json:"enabled"`
	Backend    string `json:"backend"`
	VideoCodec string `json:"video_codec"` // e.g., "libx264", "copy"
	AudioCodec string `json:"audio_codec"` // e.g., "aac", "copy"
	Preset     string `json:"preset"`      // e.g., "ultrafast", "veryfast"
	CRF        int    `json:"crf"`         // 0-51
	GOP        string `json:"gop"`         // e.g., "2s" or "60"
}

func Default() Config {
	return Config{
		ListenAddr:       ":1935",
		HTTPAddr:         ":8080",
		Upstream:         "",
		UpstreamStrategy: "round_robin",
		IdleTimeout:      Duration(30000000000), // 30 seconds in nanoseconds
		ReadBuffer:       64 * 1024,
		WriteBuffer:      64 * 1024,
	}
}

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

const (
	MinBufferSize = 4 * 1024    // 4 KB
	MaxBufferSize = 1024 * 1024 // 1 MB
)

func (c Config) Validate() error {
	if c.ListenAddr == "" {
		return errors.New("listen_addr is required")
	}
	if c.ReadBuffer <= 0 {
		return errors.New("read_buffer must be positive")
	}
	if c.WriteBuffer <= 0 {
		return errors.New("write_buffer must be positive")
	}
	if c.ReadBuffer < MinBufferSize || c.ReadBuffer > MaxBufferSize {
		return fmt.Errorf("read_buffer must be between %d and %d bytes", MinBufferSize, MaxBufferSize)
	}
	if c.WriteBuffer < MinBufferSize || c.WriteBuffer > MaxBufferSize {
		return fmt.Errorf("write_buffer must be between %d and %d bytes", MinBufferSize, MaxBufferSize)
	}
	strategy := strings.ToLower(strings.TrimSpace(c.UpstreamStrategy))
	if strategy != "" && strategy != "round_robin" && strategy != "random" {
		return errors.New("upstream_strategy must be round_robin or random")
	}
	if len(c.Upstreams) == 0 {
		if c.Upstream == "" {
			return errors.New("upstream is required")
		}
		if err := validator.ValidateUpstreamURL(c.Upstream); err != nil {
			return fmt.Errorf("upstream validation failed: %w", err)
		}
	} else {
		for i, upstream := range c.Upstreams {
			if strings.TrimSpace(upstream.URL) == "" {
				return fmt.Errorf("upstreams[%d] url is required", i)
			}
			if upstream.Weight < 0 {
				return fmt.Errorf("upstreams[%d] weight must be >= 0", i)
			}
			if err := validator.ValidateUpstreamURL(upstream.URL); err != nil {
				return fmt.Errorf("upstreams[%d] validation failed: %w", i, err)
			}
		}
	}
	if c.Security.AuthEnabled && len(c.Security.AuthTokens) == 0 {
		return errors.New("auth_enabled requires at least one auth token")
	}
	if c.Security.TLSEnabled {
		if strings.TrimSpace(c.Security.TLSCert) == "" || strings.TrimSpace(c.Security.TLSKey) == "" {
			return errors.New("tls_enabled requires tls_cert and tls_key")
		}
	}
	if c.Transcode.Enabled && strings.TrimSpace(c.Transcode.GOP) != "" {
		gop := strings.TrimSpace(c.Transcode.GOP)
		if frames, err := strconv.Atoi(gop); err == nil {
			if frames <= 0 {
				return errors.New("transcode.gop must be a positive frame count or duration")
			}
		} else if dur, err := time.ParseDuration(gop); err == nil {
			if dur <= 0 {
				return errors.New("transcode.gop must be a positive frame count or duration")
			}
		} else {
			return errors.New("transcode.gop must be a positive frame count or duration")
		}
	}
	return nil
}
