package relay

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

const (
	upstreamStrategyRoundRobin = "round_robin"
	upstreamStrategyRandom     = "random"
)

// HealthCheckConfig controls upstream health checks.
type HealthCheckConfig struct {
	Enabled  bool
	Interval time.Duration
	Timeout  time.Duration
}

// UpstreamStatus reports health and configuration for an upstream.
type UpstreamStatus struct {
	URL             string `json:"url"`
	Weight          int    `json:"weight"`
	Healthy         bool   `json:"healthy"`
	LastCheckedUnix int64  `json:"last_checked_unix"`
	LastError       string `json:"last_error,omitempty"`
}

type upstreamState struct {
	url         string
	info        UpstreamInfo
	weight      int
	healthy     bool
	lastChecked time.Time
	lastError   string
}

// UpstreamPool manages upstream selection and health.
type UpstreamPool struct {
	mu                  sync.RWMutex
	strategy            string
	endpoints           []*upstreamState
	rrIndex             int
	rng                 *rand.Rand
	healthChecksEnabled bool
}

// NewUpstreamPool builds a pool from config endpoints.
func NewUpstreamPool(endpoints []config.UpstreamEndpoint, strategy string) (*UpstreamPool, error) {
	if len(endpoints) == 0 {
		return nil, errors.New("no upstreams configured")
	}

	normalizedStrategy, err := normalizeUpstreamStrategy(strategy)
	if err != nil {
		return nil, err
	}

	pool := &UpstreamPool{
		strategy: normalizedStrategy,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, endpoint := range endpoints {
		info, err := ParseUpstream(endpoint.URL)
		if err != nil {
			return nil, err
		}
		weight := endpoint.Weight
		if weight <= 0 {
			weight = 1
		}
		pool.endpoints = append(pool.endpoints, &upstreamState{
			url:     endpoint.URL,
			info:    info,
			weight:  weight,
			healthy: true,
		})
	}

	return pool, nil
}

// Pick selects an upstream based on strategy and health.
func (p *UpstreamPool) Pick() (UpstreamInfo, string, error) {
	if p == nil {
		return UpstreamInfo{}, "", errors.New("upstream pool is nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.endpoints) == 0 {
		return UpstreamInfo{}, "", errors.New("no upstreams available")
	}

	candidates := p.healthyEndpointsLocked()
	if len(candidates) == 0 {
		candidates = p.endpoints
	}

	switch p.strategy {
	case upstreamStrategyRandom:
		return p.pickRandomLocked(candidates)
	default:
		return p.pickRoundRobinLocked(candidates)
	}
}

// StartHealthChecks begins periodic health checks.
func (p *UpstreamPool) StartHealthChecks(ctx context.Context, log *logger.Logger, cfg HealthCheckConfig) {
	if p == nil || !cfg.Enabled {
		return
	}

	cfg = normalizeHealthCheck(cfg)

	p.mu.Lock()
	p.healthChecksEnabled = true
	p.mu.Unlock()

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		p.checkAll(ctx, log, cfg.Timeout)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.checkAll(ctx, log, cfg.Timeout)
			}
		}
	}()
}

// HealthyCount returns the number of healthy upstreams.
func (p *UpstreamPool) HealthyCount() int {
	if p == nil {
		return 0
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, endpoint := range p.endpoints {
		if endpoint.healthy {
			count++
		}
	}
	return count
}

// Size returns the number of configured upstreams.
func (p *UpstreamPool) Size() int {
	if p == nil {
		return 0
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.endpoints)
}

// Strategy returns the configured selection strategy.
func (p *UpstreamPool) Strategy() string {
	if p == nil {
		return ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.strategy
}

// Stats returns current upstream status snapshots.
func (p *UpstreamPool) Stats() []UpstreamStatus {
	if p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make([]UpstreamStatus, 0, len(p.endpoints))
	for _, endpoint := range p.endpoints {
		lastChecked := int64(0)
		if !endpoint.lastChecked.IsZero() {
			lastChecked = endpoint.lastChecked.Unix()
		}
		stats = append(stats, UpstreamStatus{
			URL:             endpoint.url,
			Weight:          endpoint.weight,
			Healthy:         endpoint.healthy,
			LastCheckedUnix: lastChecked,
			LastError:       endpoint.lastError,
		})
	}
	return stats
}

func (p *UpstreamPool) healthyEndpointsLocked() []*upstreamState {
	candidates := make([]*upstreamState, 0, len(p.endpoints))
	for _, endpoint := range p.endpoints {
		if endpoint.healthy {
			candidates = append(candidates, endpoint)
		}
	}
	return candidates
}

func (p *UpstreamPool) pickRoundRobinLocked(candidates []*upstreamState) (UpstreamInfo, string, error) {
	totalWeight := 0
	for _, endpoint := range candidates {
		totalWeight += endpoint.weight
	}
	if totalWeight <= 0 {
		return UpstreamInfo{}, "", errors.New("invalid upstream weights")
	}

	pos := p.rrIndex % totalWeight
	p.rrIndex = (p.rrIndex + 1) % totalWeight

	for _, endpoint := range candidates {
		if pos < endpoint.weight {
			return endpoint.info, endpoint.url, nil
		}
		pos -= endpoint.weight
	}

	return UpstreamInfo{}, "", errors.New("no upstream selected")
}

func (p *UpstreamPool) pickRandomLocked(candidates []*upstreamState) (UpstreamInfo, string, error) {
	totalWeight := 0
	for _, endpoint := range candidates {
		totalWeight += endpoint.weight
	}
	if totalWeight <= 0 {
		return UpstreamInfo{}, "", errors.New("invalid upstream weights")
	}

	pos := p.rng.Intn(totalWeight)
	for _, endpoint := range candidates {
		if pos < endpoint.weight {
			return endpoint.info, endpoint.url, nil
		}
		pos -= endpoint.weight
	}

	return UpstreamInfo{}, "", errors.New("no upstream selected")
}

func (p *UpstreamPool) checkAll(ctx context.Context, log *logger.Logger, timeout time.Duration) {
	p.mu.RLock()
	endpoints := make([]*upstreamState, len(p.endpoints))
	copy(endpoints, p.endpoints)
	p.mu.RUnlock()

	for _, endpoint := range endpoints {
		healthy, err := probeUpstream(ctx, endpoint.info, timeout)
		p.updateHealth(endpoint, healthy, err)
		if log != nil && err != nil {
			log.Warn("upstream health check failed", "upstream", endpoint.url, "err", err)
		}
	}
}

func (p *UpstreamPool) updateHealth(endpoint *upstreamState, healthy bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	endpoint.healthy = healthy
	endpoint.lastChecked = time.Now()
	if err != nil {
		endpoint.lastError = err.Error()
	} else {
		endpoint.lastError = ""
	}
}

func normalizeUpstreamStrategy(strategy string) (string, error) {
	strategy = strings.ToLower(strings.TrimSpace(strategy))
	if strategy == "" {
		return upstreamStrategyRoundRobin, nil
	}
	switch strategy {
	case upstreamStrategyRoundRobin, upstreamStrategyRandom:
		return strategy, nil
	default:
		return "", fmt.Errorf("invalid upstream strategy %q", strategy)
	}
}

func normalizeHealthCheck(cfg HealthCheckConfig) HealthCheckConfig {
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	return cfg
}

func probeUpstream(ctx context.Context, info UpstreamInfo, timeout time.Duration) (bool, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if info.UseTLS {
		dialer := tls.Dialer{
			NetDialer: &net.Dialer{},
			Config:    &tls.Config{ServerName: info.Host},
		}
		conn, err := dialer.DialContext(dialCtx, "tcp", info.Address)
		if err != nil {
			return false, err
		}
		if closeErr := conn.Close(); closeErr != nil {
			return true, closeErr
		}
		return true, nil
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", info.Address)
	if err != nil {
		return false, err
	}
	if closeErr := conn.Close(); closeErr != nil {
		return true, closeErr
	}
	return true, nil
}
