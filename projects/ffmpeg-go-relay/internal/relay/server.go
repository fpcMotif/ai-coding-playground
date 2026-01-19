package relay

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"ffmpeg-go-relay/internal/auth"
	"ffmpeg-go-relay/internal/circuit"
	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
	"ffmpeg-go-relay/internal/metrics"
	"ffmpeg-go-relay/internal/middleware"
	"ffmpeg-go-relay/internal/pool"
	"ffmpeg-go-relay/internal/retry"
	"ffmpeg-go-relay/internal/rtmp"
	"ffmpeg-go-relay/internal/transcoder"
)

// generateRequestID creates a unique request ID for correlation
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ConnectionInfo holds information about an active connection
type ConnectionInfo struct {
	RequestID  string    `json:"request_id"`
	ClientAddr string    `json:"client_addr"`
	Upstream   string    `json:"upstream"`
	StartTime  time.Time `json:"start_time"`
	State      string    `json:"state"` // "connecting", "handshaking", "relaying", "closing"
}

// activeConnections tracks all active connections for monitoring
var activeConnections sync.Map

// GetActiveConnectionsList returns a list of all active connections
func GetActiveConnectionsList() []ConnectionInfo {
	var connections []ConnectionInfo
	activeConnections.Range(func(key, value any) bool {
		if info, ok := value.(ConnectionInfo); ok {
			connections = append(connections, info)
		}
		return true
	})
	return connections
}

// GetActiveConnectionCount returns the number of active connections
func GetActiveConnectionCount() int {
	count := 0
	activeConnections.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func trackConnectionStart(info ConnectionInfo) {
	activeConnections.Store(info.RequestID, info)
}

func updateConnectionState(requestID, state string) {
	value, ok := activeConnections.Load(requestID)
	if !ok {
		return
	}
	info, ok := value.(ConnectionInfo)
	if !ok {
		return
	}
	info.State = state
	activeConnections.Store(requestID, info)
}

func updateConnectionUpstream(requestID, upstream string) {
	value, ok := activeConnections.Load(requestID)
	if !ok {
		return
	}
	info, ok := value.(ConnectionInfo)
	if !ok {
		return
	}
	info.Upstream = upstream
	activeConnections.Store(requestID, info)
}

func trackConnectionEnd(requestID string) {
	activeConnections.Delete(requestID)
}

type Server struct {
	ListenAddr          string
	Upstream            string
	UpstreamPool        *UpstreamPool
	UpstreamHealthCheck HealthCheckConfig
	Idle                time.Duration
	ReadBuf             int
	WriteBuf            int
	Log                 *logger.Logger
	Auth                *auth.TokenAuthenticator
	RateLimit           *middleware.RateLimiter
	ConnLimit           *middleware.ConnectionLimiter
	CircuitBreaker      *circuit.Breaker
	BufPool             *pool.BytePool
	RetryConfig         retry.Config
	RetryJitter         float64
	Transcode           config.TranscodeConfig
	TLSConfig           *tls.Config
	upstreamOnce        sync.Once
	upstreamInfo        UpstreamInfo
	upstreamErr         error
}

func (s *Server) Run(ctx context.Context) error {
	var l net.Listener
	var err error
	if s.TLSConfig != nil {
		l, err = tls.Listen("tcp", s.ListenAddr, s.TLSConfig)
	} else {
		l, err = net.Listen("tcp", s.ListenAddr)
	}
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer l.Close()

	s.Log.Infof("listening on %s -> %s", s.ListenAddr, s.Upstream)

	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		l.Close()
	}()

	if s.UpstreamPool != nil && s.UpstreamHealthCheck.Enabled {
		s.UpstreamPool.StartHealthChecks(ctx, s.Log, s.UpstreamHealthCheck)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			s.Log.Errorf("accept: %v", err)
			continue
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			if err := s.handle(ctx, c); err != nil {
				s.Log.Errorf("session error: %v", err)
			}
		}(conn)
	}

	wg.Wait()
	return ctx.Err()
}

func (s *Server) handle(ctx context.Context, downstream net.Conn) (err error) {
	defer downstream.Close()

	// Generate request correlation ID for this session
	requestID := generateRequestID()
	log := s.Log.With("request_id", requestID, "client", downstream.RemoteAddr().String())

	start := time.Now()
	connInfo := ConnectionInfo{
		RequestID:  requestID,
		ClientAddr: downstream.RemoteAddr().String(),
		Upstream:   "",
		StartTime:  start,
		State:      "connecting",
	}
	trackConnectionStart(connInfo)
	defer trackConnectionEnd(requestID)

	metrics.RecordConnectionStart()
	defer func() {
		metrics.ConnectionDuration.Observe(time.Since(start).Seconds())
		if err != nil {
			metrics.RecordConnectionError()
			log.Error("session ended with error", "err", err, "duration", time.Since(start))
			return
		}
		log.Info("session completed successfully", "duration", time.Since(start))
		metrics.RecordConnectionSuccess()
	}()

	clientIP := extractIP(downstream.RemoteAddr().String())
	log.Info("new connection", "client_ip", clientIP)

	// Apply authentication if configured
	if s.Auth != nil {
		// For now, authentication would be checked via header extraction
		// In a real scenario, this would be part of the RTMP handshake
		log.Debug("auth enabled", "client_ip", clientIP)
	}

	// Apply rate limiting if configured
	if s.RateLimit != nil {
		if err = s.RateLimit.Allow(clientIP); err != nil {
			metrics.RecordRateLimitRejection()
			log.Warn("rate limit denied", "ip", clientIP, "err", err)
			return err
		}
	}

	// Apply connection limiting if configured
	if s.ConnLimit != nil {
		if err = s.ConnLimit.Acquire(clientIP); err != nil {
			metrics.RecordConnectionLimitRejection()
			log.Warn("connection limit denied", "ip", clientIP, "err", err)
			return err
		}
		defer s.ConnLimit.Release(clientIP)
	}

	dTCP, _ := downstream.(*net.TCPConn)
	if dTCP != nil {
		if err := dTCP.SetNoDelay(true); err != nil {
			log.Warn("failed to set TCP_NODELAY on downstream", "err", err)
		}
		if err := dTCP.SetReadBuffer(s.ReadBuf); err != nil {
			log.Warn("failed to set read buffer on downstream", "err", err)
		}
		if err := dTCP.SetWriteBuffer(s.WriteBuf); err != nil {
			log.Warn("failed to set write buffer on downstream", "err", err)
		}
	}

	downstream = wrapIdleConn(downstream, s.Idle)

	info, upstreamRaw, errType, selectErr := s.selectUpstream()
	if selectErr != nil {
		metrics.RecordUpstreamError(errType)
		return fmt.Errorf("%s upstream: %w", errType, selectErr)
	}
	updateConnectionUpstream(requestID, upstreamRaw)
	log = log.With("upstream", upstreamRaw)

	if s.Transcode.Enabled {
		return s.handleTranscode(ctx, downstream, log, requestID, upstreamRaw)
	}

	// Dial upstream with circuit breaker protection
	dialStart := time.Now()
	var upstream net.Conn

	dialFn := func() error {
		conn, dialErr := s.dialUpstream(ctx, info)
		if dialErr == nil {
			upstream = conn
		}
		return dialErr
	}

	if s.CircuitBreaker != nil {
		err = s.CircuitBreaker.Call(dialFn)
	} else {
		err = dialFn()
	}

	if err != nil {
		metrics.RecordUpstreamError("dial")
		return fmt.Errorf("dial upstream: %w", err)
	}
	defer upstream.Close()

	uTCP, _ := upstream.(*net.TCPConn)
	if uTCP != nil {
		if err := uTCP.SetNoDelay(true); err != nil {
			log.Warn("failed to set TCP_NODELAY on upstream", "err", err)
		}
		if err := uTCP.SetReadBuffer(s.ReadBuf); err != nil {
			log.Warn("failed to set read buffer on upstream", "err", err)
		}
		if err := uTCP.SetWriteBuffer(s.WriteBuf); err != nil {
			log.Warn("failed to set write buffer on upstream", "err", err)
		}
	}

	upstream = wrapIdleConn(upstream, s.Idle)

	updateConnectionState(requestID, "handshaking")
	if err := rtmp.ServerHandshake(downstream, nil); err != nil {
		return fmt.Errorf("downstream handshake: %w", err)
	}

	// 1. Read and inspect the CONNECT command
	log.Debug("reading connect message")
	// We use a TeeReader to buffer the exact bytes of the connect command
	// so we can replay them to the upstream if auth succeeds.
	var connectBuf bytes.Buffer
	tee := io.TeeReader(downstream, &connectBuf)
	cs := rtmp.NewChunkStream(tee)

	msg, err := cs.ReadMessage()
	if err != nil {
		log.Error("failed to read connect message", "err", err)
		return fmt.Errorf("read connect message: %w", err)
	}
	log.Debug("read connect message", "type_id", msg.Header.TypeID, "length", msg.Header.Length)

	// Decode AMF for AMF0 or AMF3 command messages.
	amfData, err := decodeConnectCommand(msg)
	if err != nil {
		return fmt.Errorf("decode amf: %w", err)
	}

	if len(amfData) < 1 {
		return fmt.Errorf("empty amf command")
	}

	cmdName, ok := amfData[0].(string)
	if !ok || cmdName != "connect" {
		return fmt.Errorf("expected 'connect' command, got %v", amfData[0])
	}

	// Extract Auth Data
	// Standard connect: ["connect", transactionID, commandObject, optionalArgs...]
	var cmdObj map[string]interface{}
	if len(amfData) >= 3 {
		cmdObj, _ = amfData[2].(map[string]interface{})
	}

	if cmdObj != nil {
		// Example: Extract 'app' or custom 'token'
		app, _ := cmdObj["app"].(string)
		tcUrl, _ := cmdObj["tcUrl"].(string)

		log.Info("rtmp connect", "app", app, "tcUrl", tcUrl)

		if s.Auth != nil {
			// Simple Auth: Check if 'app' matches a valid token
			// or if there's a specific 'token' field in the connection params
			token := app // Default usage
			if t, ok := cmdObj["token"].(string); ok {
				token = t
			}

			if err = s.Auth.Authenticate(token); err != nil {
				metrics.RecordAuthFailure()
				log.Warn("authentication failed", "token", token, "err", err)
				return fmt.Errorf("authentication failed: %w", err)
			}
		}
	} else if s.Auth != nil {
		metrics.RecordAuthFailure()
		log.Warn("authentication failed", "err", "missing command object")
		return fmt.Errorf("authentication failed: missing command object")
	}

	// 2. Connect to Upstream
	if err = rtmp.ClientHandshake(upstream, nil); err != nil {
		metrics.RecordUpstreamError("handshake")
		return fmt.Errorf("upstream handshake: %w", err)
	}
	metrics.LatencyHistogram.Observe(time.Since(dialStart).Seconds())

	log.Info("relaying", "client", connAddr(downstream), "upstream", upstreamRaw)

	// 3. Replay Connect Command
	if _, err := upstream.Write(connectBuf.Bytes()); err != nil {
		return fmt.Errorf("forward connect: %w", err)
	}

	updateConnectionState(requestID, "relaying")

	copyCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	go func() {
		buf := s.getBuffer()
		defer s.putBuffer(buf)
		_, err := io.CopyBuffer(metricsWriter{writer: upstream, direction: "upstream"}, downstream, buf)
		errCh <- err
		cancel()
	}()
	go func() {
		buf := s.getBuffer()
		defer s.putBuffer(buf)
		_, err := io.CopyBuffer(metricsWriter{writer: downstream, direction: "downstream"}, upstream, buf)
		errCh <- err
		cancel()
	}()

	// Wait for context cancellation or first error
	select {
	case <-copyCtx.Done():
	case copyErr := <-errCh:
		if copyErr != nil {
			log.Error("copy error", "err", copyErr, "direction", "first")
			err = copyErr
		}
	}

	// Drain the second goroutine's error to prevent goroutine leak
	// Use a short timeout to avoid blocking forever if goroutine is stuck
	select {
	case copyErr := <-errCh:
		if copyErr != nil && err == nil {
			log.Error("copy error", "err", copyErr, "direction", "second")
			err = copyErr
		}
	case <-time.After(100 * time.Millisecond):
		// Second goroutine didn't finish in time, it will exit when conn closes
	}

	return err
}

func (s *Server) handleTranscode(ctx context.Context, downstream net.Conn, log *logger.Logger, requestID, upstream string) error {
	// 1. Handshake (Server Side)
	// We need to act as an RTMP server to the client.
	updateConnectionState(requestID, "handshaking")
	if err := rtmp.ServerHandshake(downstream, nil); err != nil {
		return fmt.Errorf("server handshake: %w", err)
	}

	cs := rtmp.NewChunkStream(downstream)
	session := rtmp.NewServerSession(cs, downstream)

	streamName, err := session.Handshake()
	if err != nil {
		return fmt.Errorf("rtmp command handshake: %w", err)
	}
	log.Info("transcode session started", "stream", streamName)

	// 2. Start FFmpeg
	// If upstream ends with /, append streamName
	upstreamURL := upstream
	if strings.HasSuffix(upstreamURL, "/") {
		upstreamURL += streamName
	}

	tr, err := transcoder.New(ctx, s.Transcode, upstreamURL, log)
	if err != nil {
		return fmt.Errorf("start transcoder: %w", err)
	}
	defer tr.Close()

	// 3. Write FLV Header
	// We assume Audio+Video presence. In a real system, we might wait for the first A/V packets to decide.
	if err := rtmp.WriteFLVHeader(tr, true, true); err != nil {
		return fmt.Errorf("write flv header: %w", err)
	}

	updateConnectionState(requestID, "relaying")

	// 4. Relay Loop
	for {
		// Read RTMP Message
		msg, err := cs.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}
		if msg == nil {
			continue
		}

		// Convert to FLV Tag and pipe to FFmpeg
		if err := rtmp.MessageToFLVTag(tr, msg); err != nil {
			// If pipe closes, ffmpeg might have died
			return fmt.Errorf("write flv tag: %w", err)
		}
	}
}

func (s *Server) getUpstreamInfo() (UpstreamInfo, error) {
	s.upstreamOnce.Do(func() {
		s.upstreamInfo, s.upstreamErr = ParseUpstream(s.Upstream)
	})
	return s.upstreamInfo, s.upstreamErr
}

func (s *Server) selectUpstream() (UpstreamInfo, string, string, error) {
	if s.UpstreamPool != nil {
		info, raw, err := s.UpstreamPool.Pick()
		if err != nil {
			return UpstreamInfo{}, "", "select", err
		}
		return info, raw, "select", nil
	}

	info, err := s.getUpstreamInfo()
	if err != nil {
		return UpstreamInfo{}, "", "parse", err
	}
	return info, s.Upstream, "parse", nil
}

// dialUpstream dials the upstream with retry.
func (s *Server) dialUpstream(ctx context.Context, info UpstreamInfo) (net.Conn, error) {
	if s.RetryConfig.MaxAttempts <= 0 {
		return s.dialUpstreamOnce(ctx, info)
	}
	var conn net.Conn
	var err error
	dialOnce := func() error {
		c, dialErr := s.dialUpstreamOnce(ctx, info)
		if dialErr == nil {
			conn = c
		}
		return dialErr
	}
	if s.RetryJitter > 0 {
		err = retry.DoWithJitter(ctx, s.RetryConfig, s.RetryJitter, dialOnce)
	} else {
		err = retry.Do(ctx, s.RetryConfig, dialOnce)
	}
	return conn, err
}

func (s *Server) dialUpstreamOnce(ctx context.Context, info UpstreamInfo) (net.Conn, error) {
	if info.UseTLS {
		dialer := tls.Dialer{
			NetDialer: &net.Dialer{},
			Config:    &tls.Config{ServerName: info.Host},
		}
		return dialer.DialContext(ctx, "tcp", info.Address)
	}
	var dialer net.Dialer
	return dialer.DialContext(ctx, "tcp", info.Address)
}

// getBuffer gets a buffer from the pool or creates a new one
func (s *Server) getBuffer() []byte {
	if s.BufPool != nil {
		if buf := s.BufPool.Get(); buf != nil {
			return buf
		}
	}
	return make([]byte, s.ReadBuf)
}

// putBuffer returns a buffer to the pool if one exists
func (s *Server) putBuffer(buf []byte) {
	if s.BufPool != nil {
		s.BufPool.Put(buf)
	}
}

func wrapIdleConn(conn net.Conn, idle time.Duration) net.Conn {
	if conn == nil || idle <= 0 {
		return conn
	}
	return &idleConn{
		Conn: conn,
		idle: idle,
	}
}

// extractIP extracts the IP address from a remote address string
func extractIP(remoteAddr string) string {
	if remoteAddr == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	if strings.HasPrefix(remoteAddr, "[") && strings.HasSuffix(remoteAddr, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(remoteAddr, "["), "]")
	}
	return remoteAddr
}

type metricsWriter struct {
	writer    io.Writer
	direction string
}

func (m metricsWriter) Write(p []byte) (int, error) {
	if m.writer == nil {
		return 0, fmt.Errorf("nil writer")
	}
	n, err := m.writer.Write(p)
	if n > 0 {
		metrics.RecordBytesTransferred(m.direction, int64(n))
	}
	return n, err
}

type idleConn struct {
	net.Conn
	idle time.Duration
}

func (c *idleConn) Read(p []byte) (int, error) {
	if c.idle > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.idle))
	}
	return c.Conn.Read(p)
}

func (c *idleConn) Write(p []byte) (int, error) {
	if c.idle > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.idle))
	}
	return c.Conn.Write(p)
}

func decodeConnectCommand(msg *rtmp.Message) ([]interface{}, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	switch msg.Header.TypeID {
	case rtmp.TypeAMF0Command:
		return rtmp.DecodeAMF0(bytes.NewReader(msg.Payload))
	case rtmp.TypeAMF20Command:
		if len(msg.Payload) == 0 {
			return nil, fmt.Errorf("empty AMF3 payload")
		}
		if msg.Payload[0] != 0 {
			return nil, fmt.Errorf("unsupported AMF3 payload")
		}
		return rtmp.DecodeAMF0(bytes.NewReader(msg.Payload[1:]))
	default:
		return nil, fmt.Errorf("expected connect command (type %d or %d), got %d", rtmp.TypeAMF0Command, rtmp.TypeAMF20Command, msg.Header.TypeID)
	}
}

func connAddr(c net.Conn) string {
	if c == nil {
		return ""
	}
	return c.RemoteAddr().String()
}
