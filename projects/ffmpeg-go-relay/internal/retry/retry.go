package retry

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// Do retries a function with exponential backoff
func Do(ctx context.Context, cfg Config, fn func() error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 1 * time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Try the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// If this was the last attempt, don't wait
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Wait before retry
		select {
		case <-time.After(delay):
			// Continue
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		}

		// Calculate next delay with exponential backoff
		nextDelay := time.Duration(float64(delay) * cfg.Multiplier)
		if nextDelay > cfg.MaxDelay {
			nextDelay = cfg.MaxDelay
		}
		delay = nextDelay
	}

	return fmt.Errorf("max retries exceeded (%d attempts): %w", cfg.MaxAttempts, lastErr)
}

// DoWithJitter retries with exponential backoff and jitter
func DoWithJitter(ctx context.Context, cfg Config, jitterFraction float64, fn func() error) error {
	if jitterFraction < 0 || jitterFraction > 1 {
		jitterFraction = 0.1
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Add jitter
		jitter := time.Duration(float64(delay) * jitterFraction)
		jitterAmount := time.Duration(rand.Float64()*float64(jitter*2) - float64(jitter))
		actualDelay := delay + jitterAmount
		if actualDelay < 0 {
			actualDelay = delay
		}

		select {
		case <-time.After(actualDelay):
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		}

		nextDelay := time.Duration(float64(delay) * cfg.Multiplier)
		if nextDelay > cfg.MaxDelay {
			nextDelay = cfg.MaxDelay
		}
		delay = nextDelay
	}

	return fmt.Errorf("max retries exceeded (%d attempts): %w", cfg.MaxAttempts, lastErr)
}
