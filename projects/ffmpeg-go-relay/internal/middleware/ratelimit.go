package middleware

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements per-IP rate limiting using token bucket algorithm.
type RateLimiter struct {
	mu        sync.Mutex
	limiters  map[string]*rate.Limiter
	accessed  map[string]time.Time // Track last access time for cleanup
	reqPerSec float64
	burst     int
	cleanupTicker *time.Ticker
	done      chan struct{}
}

// NewRateLimiter creates a new rate limiter.
// reqPerSec: requests per second allowed per IP
// burst: maximum burst size
func NewRateLimiter(reqPerSec float64, burst int) *RateLimiter {
	if reqPerSec <= 0 {
		reqPerSec = 10 // Default 10 req/sec
	}
	if burst <= 0 {
		burst = 20 // Default burst of 20
	}

	rl := &RateLimiter{
		limiters:  make(map[string]*rate.Limiter),
		accessed:  make(map[string]time.Time),
		reqPerSec: reqPerSec,
		burst:     burst,
		done:      make(chan struct{}),
	}

	// Start cleanup goroutine to remove stale limiters
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a connection from the given IP is allowed.
// Returns nil if allowed, error if rate limit exceeded.
func (r *RateLimiter) Allow(ip string) error {
	r.mu.Lock()
	limiter, exists := r.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(r.reqPerSec), r.burst)
		r.limiters[ip] = limiter
	}
	r.accessed[ip] = time.Now()
	r.mu.Unlock()

	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded for %s", ip)
	}

	return nil
}

// GetLimiter returns the limiter for an IP (for testing/stats).
func (r *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.limiters[ip]
}

// cleanupLoop periodically removes stale rate limiters to prevent memory leaks.
func (r *RateLimiter) cleanupLoop() {
	for {
		select {
		case <-r.done:
			r.cleanupTicker.Stop()
			return
		case <-r.cleanupTicker.C:
			r.cleanup()
		}
	}
}

// cleanup removes limiters that haven't been used recently.
func (r *RateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove limiters that haven't been accessed in the last 30 minutes
	cutoffTime := time.Now().Add(-30 * time.Minute)
	for ip, lastAccess := range r.accessed {
		if lastAccess.Before(cutoffTime) {
			delete(r.limiters, ip)
			delete(r.accessed, ip)
		}
	}
}

// Stop stops the cleanup goroutine.
func (r *RateLimiter) Stop() {
	close(r.done)
}

// Stats returns statistics about current limiters (for monitoring).
func (r *RateLimiter) Stats() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return map[string]interface{}{
		"active_ips":     len(r.limiters),
		"requests_per_sec": r.reqPerSec,
		"burst_size":     r.burst,
	}
}
