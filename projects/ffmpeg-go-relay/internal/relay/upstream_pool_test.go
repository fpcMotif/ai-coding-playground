package relay

import (
	"testing"

	"ffmpeg-go-relay/internal/config"
)

func TestUpstreamPoolRoundRobin(t *testing.T) {
	pool, err := NewUpstreamPool([]config.UpstreamEndpoint{
		{URL: "rtmp://example.com/app/stream", Weight: 1},
		{URL: "rtmp://example.net/app/stream", Weight: 2},
	}, "round_robin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, raw, err := pool.Pick()
	if err != nil || raw != "rtmp://example.com/app/stream" {
		t.Fatalf("pick 1 = %q, err=%v", raw, err)
	}
	_, raw, err = pool.Pick()
	if err != nil || raw != "rtmp://example.net/app/stream" {
		t.Fatalf("pick 2 = %q, err=%v", raw, err)
	}
	_, raw, err = pool.Pick()
	if err != nil || raw != "rtmp://example.net/app/stream" {
		t.Fatalf("pick 3 = %q, err=%v", raw, err)
	}

	pool.mu.Lock()
	pool.endpoints[0].healthy = false
	pool.mu.Unlock()

	_, raw, err = pool.Pick()
	if err != nil || raw != "rtmp://example.net/app/stream" {
		t.Fatalf("pick with unhealthy upstream = %q, err=%v", raw, err)
	}
}
