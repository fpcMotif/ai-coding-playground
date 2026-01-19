package test

import (
	"context"
	"net"
	"testing"
	"time"

	"ffmpeg-go-relay/internal/circuit"
	"ffmpeg-go-relay/internal/logger"
	"ffmpeg-go-relay/internal/pool"
	"ffmpeg-go-relay/internal/relay"
	"ffmpeg-go-relay/internal/retry"
)

// BenchmarkRelayThroughput measures bytes/sec throughput
func BenchmarkRelayThroughput(b *testing.B) {
	upstream, _ := newMockUpstreamServer("127.0.0.1:0")
	upstream.start()
	defer upstream.close()

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	log := logger.New()
	server := &relay.Server{
		ListenAddr: listener.Addr().String(),
		Upstream:   upstream.addr,
		Log:        log,
		ReadBuf:    64 * 1024,
		WriteBuf:   64 * 1024,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go server.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, _ := net.Dial("tcp", listener.Addr().String())
		if client != nil {
			client.Close()
		}
	}

	cancel()
}

// BenchmarkBufferPoolAllocation measures buffer pool allocation overhead
func BenchmarkBufferPoolAllocation(b *testing.B) {
	bp := pool.New(64 * 1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		bp.Put(buf)
	}
}

// BenchmarkDirectAllocation measures direct buffer allocation
func BenchmarkDirectAllocation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = make([]byte, 64*1024)
	}
}

// BenchmarkCircuitBreakerCall measures circuit breaker overhead
func BenchmarkCircuitBreakerCall(b *testing.B) {
	breaker := circuit.New(5, 30*time.Second, 1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		breaker.Call(func() error {
			return nil
		})
	}
}

// BenchmarkRetryLogic measures retry overhead
func BenchmarkRetryLogic(b *testing.B) {
	cfg := retry.Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		retry.Do(ctx, cfg, func() error {
			return nil
		})
	}
}

// BenchmarkRelayWithPool measures relay performance with buffer pooling
func BenchmarkRelayWithPool(b *testing.B) {
	upstream, _ := newMockUpstreamServer("127.0.0.1:0")
	upstream.start()
	defer upstream.close()

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	log := logger.New()
	bufPool := pool.New(64 * 1024)

	server := &relay.Server{
		ListenAddr: listener.Addr().String(),
		Upstream:   upstream.addr,
		Log:        log,
		ReadBuf:    64 * 1024,
		WriteBuf:   64 * 1024,
		BufPool:    bufPool,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go server.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, _ := net.Dial("tcp", listener.Addr().String())
		if client != nil {
			client.Close()
		}
	}

	cancel()
}

// BenchmarkRelayWithCircuitBreaker measures circuit breaker overhead
func BenchmarkRelayWithCircuitBreaker(b *testing.B) {
	upstream, _ := newMockUpstreamServer("127.0.0.1:0")
	upstream.start()
	defer upstream.close()

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	log := logger.New()
	breaker := circuit.New(5, 30*time.Second, 1)

	server := &relay.Server{
		ListenAddr:     listener.Addr().String(),
		Upstream:       upstream.addr,
		Log:            log,
		ReadBuf:        64 * 1024,
		WriteBuf:       64 * 1024,
		CircuitBreaker: breaker,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go server.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, _ := net.Dial("tcp", listener.Addr().String())
		if client != nil {
			client.Close()
		}
	}

	cancel()
}

// BenchmarkConnectionSetup measures connection setup time
func BenchmarkConnectionSetup(b *testing.B) {
	upstream, _ := newMockUpstreamServer("127.0.0.1:0")
	upstream.start()
	defer upstream.close()

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	log := logger.New()

	server := &relay.Server{
		ListenAddr: listener.Addr().String(),
		Upstream:   upstream.addr,
		Log:        log,
		ReadBuf:    64 * 1024,
		WriteBuf:   64 * 1024,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go server.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, err := net.DialTimeout("tcp", listener.Addr().String(), 5*time.Second)
		if err == nil {
			client.Close()
		}
	}

	cancel()
}

// BenchmarkMemoryAllocation measures total memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	upstream, _ := newMockUpstreamServer("127.0.0.1:0")
	upstream.start()
	defer upstream.close()

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	log := logger.New()

	server := &relay.Server{
		ListenAddr: listener.Addr().String(),
		Upstream:   upstream.addr,
		Log:        log,
		ReadBuf:    64 * 1024,
		WriteBuf:   64 * 1024,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go server.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, _ := net.Dial("tcp", listener.Addr().String())
		if client != nil {
			client.Close()
		}
	}

	cancel()
}

// BenchmarkPoolVsAllocation compares pooling vs direct allocation
func BenchmarkPoolVsAllocation(b *testing.B) {
	bp := pool.New(64 * 1024)

	b.Run("Pool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bp.Get()
			bp.Put(buf)
		}
	})

	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 64*1024)
		}
	})
}
