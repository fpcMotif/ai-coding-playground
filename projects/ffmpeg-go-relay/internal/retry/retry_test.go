package retry

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRetrySuccess(t *testing.T) {
	cfg := DefaultConfig()

	attempts := 0
	err := Do(context.Background(), cfg, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryMaxAttemptsExceeded(t *testing.T) {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(context.Background(), cfg, func() error {
		attempts++
		return fmt.Errorf("persistent error")
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if err.Error() != "max retries exceeded (3 attempts): persistent error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRetryEventualSuccess(t *testing.T) {
	cfg := Config{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(context.Background(), cfg, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success after retries, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts before success, got %d", attempts)
	}
}

func TestRetryCancellation(t *testing.T) {
	cfg := DefaultConfig()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Do(ctx, cfg, func() error {
		return fmt.Errorf("should not be called")
	})

	if err == nil {
		t.Errorf("expected cancellation error, got nil")
	}
	if err.Error() != "retry cancelled: context canceled" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRetryExponentialBackoff(t *testing.T) {
	cfg := Config{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	startTime := time.Now()
	attempts := 0

	_ = Do(context.Background(), cfg, func() error {
		attempts++
		return fmt.Errorf("error")
	})

	elapsed := time.Since(startTime)

	// Should have backoff: 10ms + 20ms + 40ms = 70ms minimum
	// Plus some overhead for function execution
	if elapsed < 60*time.Millisecond {
		t.Errorf("expected at least 60ms delay, got %v", elapsed)
	}
}

func TestRetryMaxDelayLimit(t *testing.T) {
	cfg := Config{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     30 * time.Millisecond,
		Multiplier:   10.0, // Very high multiplier
	}

	startTime := time.Now()

	_ = Do(context.Background(), cfg, func() error {
		return fmt.Errorf("error")
	})

	elapsed := time.Since(startTime)

	// Should cap at MaxDelay: 10ms + 30ms + 30ms = 70ms minimum
	if elapsed < 60*time.Millisecond {
		t.Errorf("expected max delay to be respected, got %v", elapsed)
	}
	// But shouldn't exceed 100ms (with overhead)
	if elapsed > 150*time.Millisecond {
		t.Errorf("max delay was not respected, got %v", elapsed)
	}
}

func TestRetryDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %v", cfg.Multiplier)
	}
}

func TestRetryInvalidConfig(t *testing.T) {
	// Invalid config should use defaults
	cfg := Config{
		MaxAttempts:  0,
		InitialDelay: 0,
		MaxDelay:     0,
		Multiplier:   0,
	}

	attempts := 0
	err := Do(context.Background(), cfg, func() error {
		attempts++
		if attempts < 2 {
			return fmt.Errorf("error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success with defaults, got error: %v", err)
	}
	// Should have used default MaxAttempts of 3
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithJitter(t *testing.T) {
	cfg := Config{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := DoWithJitter(context.Background(), cfg, 0.5, func() error {
		attempts++
		return fmt.Errorf("error")
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if attempts != 4 {
		t.Errorf("expected 4 attempts, got %d", attempts)
	}
}

func TestRetryWithJitterInvalidFraction(t *testing.T) {
	cfg := DefaultConfig()

	attempts := 0
	// Invalid jitter fraction (> 1) should be clamped to 0.1
	err := DoWithJitter(context.Background(), cfg, 1.5, func() error {
		attempts++
		if attempts < 2 {
			return fmt.Errorf("error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success with clamped jitter, got error: %v", err)
	}
}

func TestRetryNegativeJitterFraction(t *testing.T) {
	cfg := DefaultConfig()

	attempts := 0
	// Negative jitter fraction should be clamped to 0.1
	err := DoWithJitter(context.Background(), cfg, -0.5, func() error {
		attempts++
		if attempts < 2 {
			return fmt.Errorf("error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success with clamped jitter, got error: %v", err)
	}
}

func TestRetryJitterPreventsNegativeDelay(t *testing.T) {
	cfg := Config{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	startTime := time.Now()
	attempts := 0

	_ = DoWithJitter(context.Background(), cfg, 1.0, func() error {
		attempts++
		return fmt.Errorf("error")
	})

	elapsed := time.Since(startTime)

	// With high jitter (1.0), delay could be negative if not handled
	// Should still take some time
	if elapsed < 5*time.Millisecond {
		t.Errorf("expected some delay even with extreme jitter, got %v", elapsed)
	}
}
