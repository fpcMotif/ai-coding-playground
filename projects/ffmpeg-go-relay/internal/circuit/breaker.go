package circuit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	Closed State = iota // Normal operation
	Open                // Failing, reject requests
	HalfOpen            // Testing if service recovered
)

// Breaker implements a circuit breaker pattern
type Breaker struct {
	mu             sync.RWMutex
	state          State
	failures       int32
	successCount   int32
	lastFailTime   time.Time
	maxFailures    int32
	resetTimeout   time.Duration
	successThresh  int32 // Successes needed in half-open to close
}

// New creates a new circuit breaker
func New(maxFailures int32, resetTimeout time.Duration, successThresh int32) *Breaker {
	if successThresh <= 0 {
		successThresh = 1
	}
	return &Breaker{
		state:         Closed,
		maxFailures:   maxFailures,
		resetTimeout:  resetTimeout,
		successThresh: successThresh,
	}
}

// Call executes a function with circuit breaker protection
func (b *Breaker) Call(fn func() error) error {
	// Phase 1: Check state and prepare (under lock)
	b.mu.Lock()
	if b.state == Open {
		if time.Since(b.lastFailTime) > b.resetTimeout {
			// Try to recover
			b.state = HalfOpen
			atomic.StoreInt32(&b.successCount, 0)
			atomic.StoreInt32(&b.failures, 0)
		} else {
			b.mu.Unlock()
			return fmt.Errorf("circuit breaker open")
		}
	}
	// Snapshot state before releasing lock
	currentState := b.state
	b.mu.Unlock()

	// Phase 2: Execute function (without lock)
	err := fn()

	// Phase 3: Record result (under lock)
	b.mu.Lock()
	defer b.mu.Unlock()

	// Re-check state hasn't been reset by another goroutine
	// If state changed while we were executing, use current state
	if b.state != currentState {
		// State changed, just return the error without modifying
		return err
	}

	if err != nil {
		return b.recordFailure(err)
	}

	return b.recordSuccess()
}

func (b *Breaker) recordFailure(err error) error {
	atomic.AddInt32(&b.failures, 1)
	b.lastFailTime = time.Now()

	if b.state == HalfOpen {
		// Failed while testing, go back to open
		b.state = Open
		return fmt.Errorf("circuit breaker open after failed recovery attempt: %w", err)
	}

	if atomic.LoadInt32(&b.failures) >= b.maxFailures {
		b.state = Open
		return fmt.Errorf("circuit breaker open after %d failures: %w", b.maxFailures, err)
	}

	return err
}

func (b *Breaker) recordSuccess() error {
	if b.state == HalfOpen {
		count := atomic.AddInt32(&b.successCount, 1)
		if count >= b.successThresh {
			b.state = Closed
			atomic.StoreInt32(&b.failures, 0)
			atomic.StoreInt32(&b.successCount, 0)
		}
		return nil
	}

	// In closed state, reset failure counter on success
	atomic.StoreInt32(&b.failures, 0)
	return nil
}

// State returns the current circuit breaker state
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Reset manually resets the circuit breaker to closed state
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = Closed
	atomic.StoreInt32(&b.failures, 0)
	atomic.StoreInt32(&b.successCount, 0)
}

// Stats returns circuit breaker statistics
func (b *Breaker) Stats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state := "closed"
	switch b.state {
	case Open:
		state = "open"
	case HalfOpen:
		state = "half-open"
	}

	return map[string]interface{}{
		"state":      state,
		"failures":   atomic.LoadInt32(&b.failures),
		"successes":  atomic.LoadInt32(&b.successCount),
		"last_fail":  b.lastFailTime.Unix(),
	}
}
