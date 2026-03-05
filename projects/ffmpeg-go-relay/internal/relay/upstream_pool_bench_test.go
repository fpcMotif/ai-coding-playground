package relay

import (
	"context"
	"net"
	"testing"
	"time"

	"ffmpeg-go-relay/internal/config"
)

func BenchmarkCheckAll(b *testing.B) {
	// Start a dummy TCP server that introduces a small delay to simulate network latency
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				time.Sleep(10 * time.Millisecond) // Simulate latency
				c.Close()
			}(conn)
		}
	}()

	addr := listener.Addr().String()

	// Create pool with 10 upstreams pointing to the dummy server
	var endpoints []config.UpstreamEndpoint
	for i := 0; i < 10; i++ {
		endpoints = append(endpoints, config.UpstreamEndpoint{
			URL:    "rtmp://" + addr + "/app/stream",
			Weight: 1,
		})
	}

	pool, err := NewUpstreamPool(endpoints, "round_robin")
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.checkAll(ctx, nil, 100*time.Millisecond)
	}
}
