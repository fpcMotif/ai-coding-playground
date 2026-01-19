package middleware

import (
	"testing"
)

func TestNewConnectionLimiter(t *testing.T) {
	cl := NewConnectionLimiter(100, 10)

	if cl == nil {
		t.Error("NewConnectionLimiter returned nil")
	}
	if cl.maxTotal != 100 {
		t.Errorf("maxTotal = %d, want 100", cl.maxTotal)
	}
	if cl.maxPerIP != 10 {
		t.Errorf("maxPerIP = %d, want 10", cl.maxPerIP)
	}
}

func TestAcquireGlobalLimit(t *testing.T) {
	cl := NewConnectionLimiter(2, 10) // Max 2 global connections

	// First two should succeed
	if err := cl.Acquire("192.168.1.1"); err != nil {
		t.Errorf("First acquire failed: %v", err)
	}

	if err := cl.Acquire("192.168.1.2"); err != nil {
		t.Errorf("Second acquire failed: %v", err)
	}

	// Third should fail (global limit)
	if err := cl.Acquire("192.168.1.3"); err == nil {
		t.Error("Third acquire should have failed (global limit)")
	}

	// Release one
	cl.Release("192.168.1.1")

	// Now we should be able to acquire again
	if err := cl.Acquire("192.168.1.3"); err != nil {
		t.Errorf("Acquire after release failed: %v", err)
	}
}

func TestAcquirePerIPLimit(t *testing.T) {
	cl := NewConnectionLimiter(100, 2) // Max 2 connections per IP

	ip := "192.168.1.1"

	// First two should succeed
	if err := cl.Acquire(ip); err != nil {
		t.Errorf("First acquire failed: %v", err)
	}

	if err := cl.Acquire(ip); err != nil {
		t.Errorf("Second acquire failed: %v", err)
	}

	// Third should fail (per-IP limit)
	if err := cl.Acquire(ip); err == nil {
		t.Error("Third acquire should have failed (per-IP limit)")
	}

	// Different IP should succeed
	if err := cl.Acquire("192.168.1.2"); err != nil {
		t.Errorf("Different IP acquire failed: %v", err)
	}
}

func TestReleasePerIP(t *testing.T) {
	cl := NewConnectionLimiter(100, 2)

	ip := "192.168.1.1"

	// Acquire twice
	cl.Acquire(ip)
	cl.Acquire(ip)

	// Release once
	cl.Release(ip)

	// Should be able to acquire again
	if err := cl.Acquire(ip); err != nil {
		t.Errorf("Acquire after release failed: %v", err)
	}
}

func TestGetActiveConnections(t *testing.T) {
	cl := NewConnectionLimiter(100, 10)

	cl.Acquire("192.168.1.1")
	cl.Acquire("192.168.1.1")
	cl.Acquire("192.168.1.2")

	total, perIP := cl.GetActiveConnections()

	if total != 3 {
		t.Errorf("Total connections = %d, want 3", total)
	}

	if len(perIP) != 2 {
		t.Errorf("Unique IPs = %d, want 2", len(perIP))
	}

	if perIP["192.168.1.1"] != 2 {
		t.Errorf("IP1 connections = %d, want 2", perIP["192.168.1.1"])
	}

	if perIP["192.168.1.2"] != 1 {
		t.Errorf("IP2 connections = %d, want 1", perIP["192.168.1.2"])
	}
}

func TestStatsOutput(t *testing.T) {
	cl := NewConnectionLimiter(100, 10)

	cl.Acquire("192.168.1.1")
	cl.Acquire("192.168.1.2")

	stats := cl.Stats()

	if stats == nil {
		t.Error("Stats returned nil")
	}

	if total, ok := stats["active_total"].(int64); !ok || total != 2 {
		t.Errorf("active_total = %v, want 2", stats["active_total"])
	}

	if maxTotal, ok := stats["max_total"].(int64); !ok || maxTotal != 100 {
		t.Errorf("max_total = %v, want 100", stats["max_total"])
	}

	if maxPerIP, ok := stats["max_per_ip"].(int64); !ok || maxPerIP != 10 {
		t.Errorf("max_per_ip = %v, want 10", stats["max_per_ip"])
	}
}

func TestUnlimitedGlobal(t *testing.T) {
	cl := NewConnectionLimiter(0, 2) // No global limit

	// Should be able to acquire many
	for i := 0; i < 10; i++ {
		if err := cl.Acquire("192.168.1.1"); err == nil || i >= 2 {
			if i >= 2 && err == nil {
				t.Errorf("Iteration %d should fail (per-IP limit)", i)
			}
		}
	}
}

func TestUnlimitedPerIP(t *testing.T) {
	cl := NewConnectionLimiter(10, 0) // No per-IP limit

	ip := "192.168.1.1"

	// Should be able to acquire up to global limit
	for i := 0; i < 10; i++ {
		if err := cl.Acquire(ip); err != nil {
			t.Errorf("Iteration %d failed: %v", i, err)
		}
	}

	// 11th should fail (global limit)
	if err := cl.Acquire(ip); err == nil {
		t.Error("11th acquire should fail (global limit)")
	}
}

func TestConcurrentAcquire(t *testing.T) {
	cl := NewConnectionLimiter(100, 100)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			cl.Acquire("192.168.1.1")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	total, _ := cl.GetActiveConnections()
	if total != 10 {
		t.Errorf("Total after concurrent acquire = %d, want 10", total)
	}
}
