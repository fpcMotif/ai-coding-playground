package middleware

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ConnectionLimiter enforces connection limits (global and per-IP).
type ConnectionLimiter struct {
	mu              sync.RWMutex
	activePerIP     map[string]*atomic.Int64
	activeTotal     atomic.Int64
	maxTotal        int64
	maxPerIP        int64
}

// NewConnectionLimiter creates a new connection limiter.
// maxTotal: maximum total connections (0 = unlimited)
// maxPerIP: maximum connections per IP (0 = unlimited)
func NewConnectionLimiter(maxTotal, maxPerIP int64) *ConnectionLimiter {
	return &ConnectionLimiter{
		activePerIP: make(map[string]*atomic.Int64),
		maxTotal:    maxTotal,
		maxPerIP:    maxPerIP,
	}
}

// Acquire attempts to acquire a connection slot for the given IP.
// Returns nil if acquired, error if limits exceeded.
// Uses atomic CompareAndSwap to prevent TOCTOU race conditions.
func (c *ConnectionLimiter) Acquire(ip string) error {
	// Atomically check and increment global limit
	if c.maxTotal > 0 {
		for {
			current := c.activeTotal.Load()
			if current >= c.maxTotal {
				return fmt.Errorf("global connection limit exceeded (%d)", c.maxTotal)
			}
			if c.activeTotal.CompareAndSwap(current, current+1) {
				break
			}
		}
	} else {
		c.activeTotal.Add(1)
	}

	// Atomically check and increment per-IP limit
	if c.maxPerIP > 0 {
		ipCounter := c.getOrCreateCounter(ip)
		for {
			current := ipCounter.Load()
			if current >= c.maxPerIP {
				// Rollback global counter since we failed per-IP check
				c.activeTotal.Add(-1)
				return fmt.Errorf("per-IP connection limit exceeded for %s (%d)", ip, c.maxPerIP)
			}
			if ipCounter.CompareAndSwap(current, current+1) {
				break
			}
		}
	}

	return nil
}

// Release releases a connection slot for the given IP.
// Cleans up zero-count entries to prevent memory leak.
func (c *ConnectionLimiter) Release(ip string) {
	if c.maxPerIP > 0 {
		ipCounter := c.getOrCreateCounter(ip)
		newCount := ipCounter.Add(-1)

		// Clean up zero-count entries to prevent memory leak
		if newCount <= 0 {
			c.mu.Lock()
			// Double-check under lock to avoid race
			if ipCounter.Load() <= 0 {
				delete(c.activePerIP, ip)
			}
			c.mu.Unlock()
		}
	}

	c.activeTotal.Add(-1)
}

// getOrCreateCounter gets or creates the counter for an IP.
func (c *ConnectionLimiter) getOrCreateCounter(ip string) *atomic.Int64 {
	c.mu.RLock()
	counter, exists := c.activePerIP[ip]
	c.mu.RUnlock()

	if exists {
		return counter
	}

	// Create new counter
	counter = &atomic.Int64{}
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check pattern
	if existing, ok := c.activePerIP[ip]; ok {
		return existing
	}

	c.activePerIP[ip] = counter
	return counter
}

// GetActiveConnections returns current active connection counts.
func (c *ConnectionLimiter) GetActiveConnections() (total int64, perIP map[string]int64) {
	total = c.activeTotal.Load()

	c.mu.RLock()
	defer c.mu.RUnlock()

	perIP = make(map[string]int64)
	for ip, counter := range c.activePerIP {
		if count := counter.Load(); count > 0 {
			perIP[ip] = count
		}
	}

	return total, perIP
}

// Stats returns statistics about connection limits.
func (c *ConnectionLimiter) Stats() map[string]interface{} {
	total, perIP := c.GetActiveConnections()

	return map[string]interface{}{
		"active_total":    total,
		"active_per_ip":   perIP,
		"max_total":       c.maxTotal,
		"max_per_ip":      c.maxPerIP,
		"unique_ips":      len(perIP),
	}
}
