package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Define all Prometheus metrics
var (
	// Active connections gauge
	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rtmp_relay_active_connections",
		Help: "Number of active RTMP relay connections",
	})

	// Total connections counter
	TotalConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rtmp_relay_connections_total",
		Help: "Total number of RTMP connections",
	}, []string{"status"})

	// Bytes transferred counter
	BytesTransferred = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rtmp_relay_bytes_total",
		Help: "Total bytes transferred",
	}, []string{"direction"})

	// Connection duration histogram
	ConnectionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "rtmp_relay_connection_duration_seconds",
		Help:    "Connection duration in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to 512s
	})

	// Upstream connection latency histogram
	LatencyHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "rtmp_relay_latency_seconds",
		Help:    "Relay latency in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Upstream errors counter
	UpstreamErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rtmp_relay_upstream_errors_total",
		Help: "Total upstream connection errors",
	}, []string{"error_type"})

	// Rate limit rejections counter
	RateLimitRejections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rtmp_relay_rate_limit_rejections_total",
		Help: "Total connections rejected by rate limiting",
	})

	// Connection limit rejections counter
	ConnectionLimitRejections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rtmp_relay_connection_limit_rejections_total",
		Help: "Total connections rejected by connection limits",
	})

	// Authentication failures counter
	AuthFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rtmp_relay_auth_failures_total",
		Help: "Total authentication failures",
	})
)

// RecordConnectionStart records when a connection starts
func RecordConnectionStart() {
	ActiveConnections.Inc()
	TotalConnections.WithLabelValues("started").Inc()
}

// RecordConnectionSuccess records when a connection completes successfully
func RecordConnectionSuccess() {
	ActiveConnections.Dec()
	TotalConnections.WithLabelValues("success").Inc()
}

// RecordConnectionError records when a connection ends with error
func RecordConnectionError() {
	ActiveConnections.Dec()
	TotalConnections.WithLabelValues("error").Inc()
}

// RecordBytesTransferred records bytes transferred in a direction
func RecordBytesTransferred(direction string, bytes int64) {
	BytesTransferred.WithLabelValues(direction).Add(float64(bytes))
}

// RecordUpstreamError records an upstream error
func RecordUpstreamError(errorType string) {
	UpstreamErrors.WithLabelValues(errorType).Inc()
}

// RecordRateLimitRejection records a rate limit rejection
func RecordRateLimitRejection() {
	RateLimitRejections.Inc()
}

// RecordConnectionLimitRejection records a connection limit rejection
func RecordConnectionLimitRejection() {
	ConnectionLimitRejections.Inc()
}

// RecordAuthFailure records an authentication failure
func RecordAuthFailure() {
	AuthFailures.Inc()
}
