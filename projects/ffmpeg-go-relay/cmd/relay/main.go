package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ffmpeg-go-relay/internal/auth"
	"ffmpeg-go-relay/internal/circuit"
	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/httpserver"
	"ffmpeg-go-relay/internal/logger"
	"ffmpeg-go-relay/internal/middleware"
	"ffmpeg-go-relay/internal/pool"
	"ffmpeg-go-relay/internal/relay"
	"ffmpeg-go-relay/internal/retry"
)

func main() {
	cfgPath := flag.String("config", "", "Path to JSON config file")
	listen := flag.String("listen", "", "Listen address (overrides config)")
	httpAddr := flag.String("http-addr", "", "HTTP listen address for health/metrics (empty to disable)")
	upstream := flag.String("upstream", "", "Upstream RTMP endpoint (e.g., rtmp://host/app/stream)")
	idle := flag.Duration("idle-timeout", 0, "Idle timeout for connections (e.g., 30s)")
	readBuf := flag.Int("read-buffer", 64*1024, "Read buffer size in bytes")
	writeBuf := flag.Int("write-buffer", 64*1024, "Write buffer size in bytes")
	flag.Parse()

	log := logger.New()

	baseCfg := config.Default()
	if *cfgPath != "" {
		loaded, err := config.LoadFile(*cfgPath)
		if err != nil {
			log.Fatal("failed to load config", "err", err)
		}
		baseCfg = loaded
	}

	if *listen != "" {
		baseCfg.ListenAddr = *listen
	}
	if *httpAddr != "" {
		baseCfg.HTTPAddr = *httpAddr
	}
	if *upstream != "" {
		baseCfg.Upstream = *upstream
	}
	if *idle > 0 {
		baseCfg.IdleTimeout = config.Duration(*idle)
	}
	if *readBuf > 0 {
		baseCfg.ReadBuffer = *readBuf
	}
	if *writeBuf > 0 {
		baseCfg.WriteBuffer = *writeBuf
	}

	if err := baseCfg.Validate(); err != nil {
		log.Fatal("invalid config", "err", err)
	}

	upstreamEndpoints := baseCfg.Upstreams
	if len(upstreamEndpoints) == 0 && baseCfg.Upstream != "" {
		upstreamEndpoints = []config.UpstreamEndpoint{
			{URL: baseCfg.Upstream},
		}
	}
	upstreamPool, err := relay.NewUpstreamPool(upstreamEndpoints, baseCfg.UpstreamStrategy)
	if err != nil {
		log.Fatal("invalid upstream configuration", "err", err)
	}

	primaryUpstream := baseCfg.Upstream
	if primaryUpstream == "" && len(upstreamEndpoints) > 0 {
		primaryUpstream = upstreamEndpoints[0].URL
	}

	upstreamHealthCheck := relay.HealthCheckConfig{
		Enabled:  baseCfg.UpstreamHealthCheck.Enabled,
		Interval: time.Duration(baseCfg.UpstreamHealthCheck.IntervalSec) * time.Second,
		Timeout:  time.Duration(baseCfg.UpstreamHealthCheck.TimeoutSec) * time.Second,
	}

	var authenticator *auth.TokenAuthenticator
	if baseCfg.Security.AuthEnabled {
		authenticator = auth.NewTokenAuthenticator(baseCfg.Security.AuthTokens)
	}

	var tlsConfig *tls.Config
	if baseCfg.Security.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(baseCfg.Security.TLSCert, baseCfg.Security.TLSKey)
		if err != nil {
			log.Fatal("failed to load TLS key pair", "err", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	var rateLimiter *middleware.RateLimiter
	if baseCfg.RateLimit.Enabled {
		rateLimiter = middleware.NewRateLimiter(baseCfg.RateLimit.RequestsPerSec, baseCfg.RateLimit.Burst)
		defer rateLimiter.Stop()
	}

	var connLimiter *middleware.ConnectionLimiter
	if baseCfg.ConnectionLimit.MaxTotal > 0 || baseCfg.ConnectionLimit.MaxPerIP > 0 {
		connLimiter = middleware.NewConnectionLimiter(baseCfg.ConnectionLimit.MaxTotal, baseCfg.ConnectionLimit.MaxPerIP)
	}

	var breaker *circuit.Breaker
	if baseCfg.CircuitBreaker.Enabled {
		resetTimeout := time.Duration(baseCfg.CircuitBreaker.ResetTimeoutSec) * time.Second
		if resetTimeout <= 0 {
			resetTimeout = 30 * time.Second
		}
		maxFailures := baseCfg.CircuitBreaker.MaxFailures
		if maxFailures <= 0 {
			maxFailures = 5
		}
		successThresh := baseCfg.CircuitBreaker.SuccessThresh
		if successThresh <= 0 {
			successThresh = 1
		}
		breaker = circuit.New(maxFailures, resetTimeout, successThresh)
	}

	retryCfg := retry.Config{}
	retryJitter := 0.0
	if baseCfg.Retry.Enabled {
		retryCfg = retry.Config{
			MaxAttempts:  baseCfg.Retry.MaxAttempts,
			InitialDelay: time.Duration(baseCfg.Retry.InitialDelaySec) * time.Second,
			MaxDelay:     time.Duration(baseCfg.Retry.MaxDelaySec) * time.Second,
			Multiplier:   baseCfg.Retry.Multiplier,
		}
		retryJitter = baseCfg.Retry.JitterFraction
	}

	bufPool := pool.New(baseCfg.ReadBuffer)

	srv := relay.Server{
		ListenAddr:          baseCfg.ListenAddr,
		Upstream:            primaryUpstream,
		Idle:                time.Duration(baseCfg.IdleTimeout),
		ReadBuf:             baseCfg.ReadBuffer,
		WriteBuf:            baseCfg.WriteBuffer,
		Log:                 log,
		Auth:                authenticator,
		RateLimit:           rateLimiter,
		ConnLimit:           connLimiter,
		CircuitBreaker:      breaker,
		BufPool:             bufPool,
		RetryConfig:         retryCfg,
		RetryJitter:         retryJitter,
		Transcode:           baseCfg.Transcode,
		TLSConfig:           tlsConfig,
		UpstreamPool:        upstreamPool,
		UpstreamHealthCheck: upstreamHealthCheck,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if baseCfg.HTTPAddr != "" {
		httpSrv := httpserver.New(baseCfg.HTTPAddr, log, &httpserver.RelayStats{
			ConnLimiter:    connLimiter,
			RateLimit:      rateLimiter,
			Upstream:       primaryUpstream,
			UpstreamPool:   upstreamPool,
			CircuitBreaker: breaker,
			BufferPool:     bufPool,
		}, tlsConfig)
		go func() {
			if err := httpSrv.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Error("http server error", "err", err)
			}
		}()
	}

	errs := make(chan error, 1)
	go func() {
		errs <- srv.Run(ctx)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down", "reason", ctx.Err())
	case err := <-errs:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}

	// Graceful shutdown with connection draining
	drainTimeout := 10 * time.Second
	drainInterval := time.Second
	drainStart := time.Now()

	log.Info("draining connections", "timeout", drainTimeout)

	for {
		elapsed := time.Since(drainStart)
		if elapsed >= drainTimeout {
			log.Warn("drain timeout reached, forcing shutdown", "elapsed", elapsed)
			break
		}

		// Check remaining connections if connection limiter is available
		if connLimiter != nil {
			total, _ := connLimiter.GetActiveConnections()
			if total == 0 {
				log.Info("all connections drained", "elapsed", elapsed)
				break
			}
			log.Info("waiting for connections to close", "active", total, "elapsed", elapsed, "remaining", drainTimeout-elapsed)
		} else {
			// No connection tracking, just wait a bit
			time.Sleep(drainInterval)
			break
		}

		time.Sleep(drainInterval)
	}

	log.Info("shutdown complete", "total_drain_time", time.Since(drainStart))
}
