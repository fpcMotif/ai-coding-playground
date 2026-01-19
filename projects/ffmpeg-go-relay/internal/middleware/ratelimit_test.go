package middleware

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 20)
	defer rl.Stop()

	if rl == nil {
		t.Error("NewRateLimiter returned nil")
	}
	if rl.reqPerSec != 10 {
		t.Errorf("reqPerSec = %v, want 10", rl.reqPerSec)
	}
	if rl.burst != 20 {
		t.Errorf("burst = %d, want 20", rl.burst)
	}
}

func TestRateLimitAllow(t *testing.T) {
	rl := NewRateLimiter(2, 2) // 2 req/sec, burst of 2
	defer rl.Stop()

	// First two requests should succeed (burst)
	if err := rl.Allow("192.168.1.1"); err != nil {
		t.Errorf("First request failed: %v", err)
	}

	if err := rl.Allow("192.168.1.1"); err != nil {
		t.Errorf("Second request failed: %v", err)
	}

	// Third request should fail (burst exhausted)
	if err := rl.Allow("192.168.1.1"); err == nil {
		t.Error("Third request should have failed")
	}

	// Wait for token to refill
	time.Sleep(600 * time.Millisecond)

	// Next request should succeed
	if err := rl.Allow("192.168.1.1"); err != nil {
		t.Errorf("Request after refill failed: %v", err)
	}
}

func TestRateLimitPerIP(t *testing.T) {
	rl := NewRateLimiter(1, 1) // 1 req/sec, burst of 1
	defer rl.Stop()

	// IP1: should be allowed
	if err := rl.Allow("192.168.1.1"); err != nil {
		t.Errorf("IP1 request failed: %v", err)
	}

	// IP2: should be allowed (different IP)
	if err := rl.Allow("192.168.1.2"); err != nil {
		t.Errorf("IP2 request failed: %v", err)
	}

	// IP1 again: should fail (burst exhausted)
	if err := rl.Allow("192.168.1.1"); err == nil {
		t.Error("IP1 second request should have failed")
	}

	// IP2 again: should fail (burst exhausted)
	if err := rl.Allow("192.168.1.2"); err == nil {
		t.Error("IP2 second request should have failed")
	}
}

func TestRateLimiterStats(t *testing.T) {
	rl := NewRateLimiter(10, 20)
	defer rl.Stop()

	_ = rl.Allow("192.168.1.1")
	_ = rl.Allow("192.168.1.2")

	stats := rl.Stats()
	if stats == nil {
		t.Error("Stats returned nil")
	}

	if active, ok := stats["active_ips"].(int); !ok || active != 2 {
		t.Errorf("active_ips = %v, want 2", stats["active_ips"])
	}
}

func TestRateLimiterGetLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 20)
	defer rl.Stop()

	_ = rl.Allow("192.168.1.1")

	limiter := rl.GetLimiter("192.168.1.1")
	if limiter == nil {
		t.Error("GetLimiter returned nil")
	}
}

func TestRateLimiterStop(t *testing.T) {
	rl := NewRateLimiter(10, 20)
	rl.Stop() // Should not panic

	// After stop, cleanup loop should exit
	time.Sleep(100 * time.Millisecond)
}

func TestDefaultValues(t *testing.T) {
	rl := NewRateLimiter(0, 0) // Test with invalid values
	defer rl.Stop()

	if rl.reqPerSec != 10 {
		t.Errorf("Default reqPerSec = %v, want 10", rl.reqPerSec)
	}
	if rl.burst != 20 {
		t.Errorf("Default burst = %d, want 20", rl.burst)
	}
}
