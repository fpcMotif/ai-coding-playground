package httpserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"ffmpeg-go-relay/internal/circuit"
	"ffmpeg-go-relay/internal/logger"
	"ffmpeg-go-relay/internal/middleware"
	"ffmpeg-go-relay/internal/pool"
	"ffmpeg-go-relay/internal/relay"
)

// Build information, set at compile time via -ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Server provides HTTP endpoints for health checks and metrics.
type Server struct {
	addr        string
	log         *logger.Logger
	server      *http.Server
	relayStats  *RelayStats
	startedAt   time.Time
	enablePprof bool
	tlsConfig   *tls.Config
}

// RelayStats holds references to relay state for stats reporting.
type RelayStats struct {
	ConnLimiter    *middleware.ConnectionLimiter
	RateLimit      *middleware.RateLimiter
	CircuitBreaker *circuit.Breaker
	BufferPool     *pool.BytePool
	Upstream       string
	UpstreamPool   *relay.UpstreamPool
}

// New creates a new HTTP server.
func New(addr string, log *logger.Logger, stats *RelayStats, tlsConfig *tls.Config) *Server {
	return &Server{
		addr:        addr,
		log:         log,
		relayStats:  stats,
		startedAt:   time.Now(),
		enablePprof: false, // Disabled by default for security
		tlsConfig:   tlsConfig,
	}
}

// NewWithPprof creates a new HTTP server with pprof enabled.
func NewWithPprof(addr string, log *logger.Logger, stats *RelayStats, enablePprof bool, tlsConfig *tls.Config) *Server {
	return &Server{
		addr:        addr,
		log:         log,
		relayStats:  stats,
		startedAt:   time.Now(),
		enablePprof: enablePprof,
		tlsConfig:   tlsConfig,
	}
}

// Run starts the HTTP server and blocks until context is done.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	// Root endpoint
	mux.HandleFunc("/", s.handleRoot)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Readiness check endpoint
	mux.HandleFunc("/ready", s.handleReady)

	// Liveness check endpoint
	mux.HandleFunc("/livez", s.handleLivez)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Status endpoint
	mux.HandleFunc("/status", s.handleStatus)

	// Version endpoint
	mux.HandleFunc("/version", s.handleVersion)

	// Admin endpoints
	mux.HandleFunc("/admin/connections", s.handleAdminConnections)
	mux.HandleFunc("/admin/circuit-breaker", s.handleAdminCircuitBreaker)
	mux.HandleFunc("/admin/circuit-breaker/reset", s.handleAdminCircuitBreakerReset)

	// Performance profiling endpoints (pprof) - only if enabled
	if s.enablePprof {
		s.log.Warn("pprof profiling endpoints enabled - do not expose in production!")
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	}

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Start listening
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server starting", "addr", s.addr)
		if s.tlsConfig != nil {
			s.server.TLSConfig = s.tlsConfig
			errCh <- s.server.ListenAndServeTLS("", "")
			return
		}
		errCh <- s.server.ListenAndServe()
	}()

	// Wait for context done or error
	select {
	case <-ctx.Done():
		s.log.Info("http server shutdown initiated")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server error: %w", err)
		}
		return nil
	}
}

// handleRoot provides a friendly root endpoint.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"service": "ffmpeg-go-relay",
		"message": "relay is live",
		"time":    time.Now().Unix(),
	}); err != nil {
		s.log.Error("failed to encode root response", "err", err)
	}
}

// handleHealth checks if server is running.
// Returns 200 if running, 503 if not.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "healthy",
		"time":   time.Now().Unix(),
	}); err != nil {
		s.log.Error("failed to encode health response", "err", err)
	}
}

// handleReady checks if server is ready to accept connections.
// Includes checks for upstream connectivity.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check upstream connectivity with timeout
	timeoutCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	upstream := ""
	if s.relayStats != nil {
		upstream = s.relayStats.Upstream
	}

	upstreamReachable := true
	if s.relayStats != nil && s.relayStats.UpstreamPool != nil {
		healthy := s.relayStats.UpstreamPool.HealthyCount()
		upstreamReachable = healthy > 0
	} else if upstream != "" {
		info, err := relay.ParseUpstream(upstream)
		if err != nil {
			upstreamReachable = false
		} else {
			dialer := &net.Dialer{}
			var conn net.Conn
			if info.UseTLS {
				tlsDialer := tls.Dialer{
					NetDialer: dialer,
					Config:    &tls.Config{ServerName: info.Host},
				}
				conn, err = tlsDialer.DialContext(timeoutCtx, "tcp", info.Address)
			} else {
				conn, err = dialer.DialContext(timeoutCtx, "tcp", info.Address)
			}
			if err != nil {
				upstreamReachable = false
			} else if closeErr := conn.Close(); closeErr != nil {
				s.log.Warn("failed to close readiness check connection", "err", closeErr)
			}
		}
	}

	response := map[string]any{
		"ready":     upstreamReachable,
		"time":      time.Now().Unix(),
		"upstream":  upstream,
		"reachable": upstreamReachable,
	}

	if s.relayStats != nil && s.relayStats.UpstreamPool != nil {
		response["upstreams_total"] = s.relayStats.UpstreamPool.Size()
		response["upstreams_healthy"] = s.relayStats.UpstreamPool.HealthyCount()
	}

	if !upstreamReachable {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("failed to encode ready response", "err", err)
	}
}

// handleLivez checks if server process is alive (always returns 200).
func (s *Server) handleLivez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"alive": true,
		"time":  time.Now().Unix(),
	}); err != nil {
		s.log.Error("failed to encode livez response", "err", err)
	}
}

// handleStatus returns detailed status information.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	upstream := ""
	if s.relayStats != nil {
		upstream = s.relayStats.Upstream
	}

	status := map[string]any{
		"time":           time.Now().Unix(),
		"started_at":     s.startedAt.Unix(),
		"uptime_seconds": time.Since(s.startedAt).Seconds(),
		"upstream":       upstream,
	}

	if s.relayStats != nil && s.relayStats.UpstreamPool != nil {
		status["upstreams"] = s.relayStats.UpstreamPool.Stats()
		status["upstream_strategy"] = s.relayStats.UpstreamPool.Strategy()
	}

	if s.relayStats != nil && s.relayStats.ConnLimiter != nil {
		status["connections"] = s.relayStats.ConnLimiter.Stats()
	}

	if s.relayStats != nil && s.relayStats.RateLimit != nil {
		status["rate_limit"] = s.relayStats.RateLimit.Stats()
	}

	if s.relayStats != nil && s.relayStats.CircuitBreaker != nil {
		status["circuit_breaker"] = s.relayStats.CircuitBreaker.Stats()
	}

	if s.relayStats != nil && s.relayStats.BufferPool != nil {
		status["buffer_pool"] = s.relayStats.BufferPool.Stats()
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.log.Error("failed to encode status response", "err", err)
	}
}

// handleVersion returns build version information.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"version":    Version,
		"git_commit": GitCommit,
		"build_time": BuildTime,
		"go_version": runtime.Version(),
	}); err != nil {
		s.log.Error("failed to encode version response", "err", err)
	}
}

// handleAdminConnections returns information about active connections.
func (s *Server) handleAdminConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"error": "method not allowed",
		}); err != nil {
			s.log.Error("failed to encode admin connections error response", "err", err)
		}
		return
	}

	// Get detailed connection list from relay package
	connections := relay.GetActiveConnectionsList()

	response := map[string]any{
		"time":              time.Now().Unix(),
		"total_connections": len(connections),
		"connections":       connections,
	}

	// Also include per-IP stats if available
	if s.relayStats != nil && s.relayStats.ConnLimiter != nil {
		_, perIP := s.relayStats.ConnLimiter.GetActiveConnections()
		response["connections_per_ip"] = perIP
		response["unique_ips"] = len(perIP)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("failed to encode admin connections response", "err", err)
	}
}

// handleAdminCircuitBreaker returns circuit breaker state.
func (s *Server) handleAdminCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"error": "method not allowed",
		}); err != nil {
			s.log.Error("failed to encode circuit breaker error response", "err", err)
		}
		return
	}

	response := map[string]any{
		"time": time.Now().Unix(),
	}

	if s.relayStats != nil && s.relayStats.CircuitBreaker != nil {
		response["circuit_breaker"] = s.relayStats.CircuitBreaker.Stats()
		response["available"] = true
	} else {
		response["circuit_breaker"] = nil
		response["available"] = false
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("failed to encode circuit breaker response", "err", err)
	}
}

// handleAdminCircuitBreakerReset resets the circuit breaker.
func (s *Server) handleAdminCircuitBreakerReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"error": "method not allowed, use POST",
		}); err != nil {
			s.log.Error("failed to encode circuit breaker reset error response", "err", err)
		}
		return
	}

	if s.relayStats == nil || s.relayStats.CircuitBreaker == nil {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"error": "circuit breaker not configured",
		}); err != nil {
			s.log.Error("failed to encode circuit breaker not found response", "err", err)
		}
		return
	}

	s.relayStats.CircuitBreaker.Reset()
	s.log.Info("circuit breaker manually reset via admin API")

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "circuit breaker reset to closed state",
		"time":    time.Now().Unix(),
	}); err != nil {
		s.log.Error("failed to encode circuit breaker reset response", "err", err)
	}
}
