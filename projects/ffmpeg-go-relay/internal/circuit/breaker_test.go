package circuit

import (
	"fmt"
	"testing"
	"time"
)

func TestBreakerNewDefaults(t *testing.T) {
	b := New(5, 30*time.Second, 1)
	if b.state != Closed {
		t.Errorf("expected initial state Closed, got %v", b.state)
	}
	if b.maxFailures != 5 {
		t.Errorf("expected maxFailures 5, got %d", b.maxFailures)
	}
	if b.successThresh != 1 {
		t.Errorf("expected successThresh 1, got %d", b.successThresh)
	}
}

func TestBreakerClosedState(t *testing.T) {
	b := New(3, 30*time.Second, 1)

	// Successful calls in closed state should succeed
	err := b.Call(func() error { return nil })
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if b.State() != Closed {
		t.Errorf("expected state to remain Closed, got %v", b.State())
	}
}

func TestBreakerOpenAfterFailures(t *testing.T) {
	b := New(2, 30*time.Second, 1)

	// First failure
	err := b.Call(func() error { return fmt.Errorf("test error") })
	if err == nil || err.Error() != "test error" {
		t.Errorf("expected test error, got: %v", err)
	}
	if b.State() != Closed {
		t.Errorf("expected state Closed after 1 failure, got %v", b.State())
	}

	// Second failure - should open
	err = b.Call(func() error { return fmt.Errorf("test error 2") })
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if b.State() != Open {
		t.Errorf("expected state Open after 2 failures, got %v", b.State())
	}

	// Circuit should be open, next call should be rejected
	err = b.Call(func() error { return nil })
	if err == nil || err.Error() != "circuit breaker open" {
		t.Errorf("expected 'circuit breaker open' error, got: %v", err)
	}
}

func TestBreakerHalfOpenRecovery(t *testing.T) {
	resetTimeout := 50 * time.Millisecond
	b := New(1, resetTimeout, 1)

	// Trigger open state
	_ = b.Call(func() error { return fmt.Errorf("fail") })

	if b.State() != Open {
		t.Errorf("expected Open, got %v", b.State())
	}

	// Wait for reset timeout
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// Next call should transition to HalfOpen
	err := b.Call(func() error { return nil })
	if err != nil {
		t.Errorf("expected success in HalfOpen, got error: %v", err)
	}

	// Should be back to Closed after successful call
	if b.State() != Closed {
		t.Errorf("expected state Closed after recovery, got %v", b.State())
	}
}

func TestBreakerHalfOpenFailure(t *testing.T) {
	resetTimeout := 50 * time.Millisecond
	b := New(1, resetTimeout, 1)

	// Trigger open state
	_ = b.Call(func() error { return fmt.Errorf("fail") })

	// Wait for reset timeout
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// Fail in HalfOpen should go back to Open
	err := b.Call(func() error { return fmt.Errorf("recovery failed") })
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if b.State() != Open {
		t.Errorf("expected state Open after HalfOpen failure, got %v", b.State())
	}
}

func TestBreakerStats(t *testing.T) {
	b := New(3, 30*time.Second, 1)

	// Record some failures
	b.Call(func() error { return fmt.Errorf("fail") })
	b.Call(func() error { return fmt.Errorf("fail") })

	stats := b.Stats()
	if stats["state"] != "closed" {
		t.Errorf("expected state 'closed', got %v", stats["state"])
	}
	if stats["failures"] != int32(2) {
		t.Errorf("expected failures 2, got %v", stats["failures"])
	}
}

func TestBreakerSuccessThreshold(t *testing.T) {
	resetTimeout := 50 * time.Millisecond
	b := New(1, resetTimeout, 2) // Need 2 successes to close

	// Trigger open
	_ = b.Call(func() error { return fmt.Errorf("fail") })

	// Wait and transition to HalfOpen
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// First success in HalfOpen
	err := b.Call(func() error { return nil })
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if b.State() != HalfOpen {
		t.Errorf("expected HalfOpen after 1 success with threshold 2, got %v", b.State())
	}

	// Second success should close
	err = b.Call(func() error { return nil })
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if b.State() != Closed {
		t.Errorf("expected Closed after 2 successes, got %v", b.State())
	}
}

func TestBreakerResetFailureCounter(t *testing.T) {
	b := New(3, 30*time.Second, 1)

	// Record a failure
	b.Call(func() error { return fmt.Errorf("fail") })

	// Success should reset counter
	b.Call(func() error { return nil })

	stats := b.Stats()
	if stats["failures"] != int32(0) {
		t.Errorf("expected failures 0 after success in Closed, got %v", stats["failures"])
	}
}
