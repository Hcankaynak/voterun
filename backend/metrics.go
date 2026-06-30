package main

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus collectors for VoteRun. The default registry used by promauto also
// exposes Go runtime and process metrics (goroutines, GC, memory, CPU).
var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests handled, by method, route and status.",
	}, []string{"method", "path", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds, by method, route and status.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being served.",
	})

	wsActiveConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_active_connections",
		Help: "Number of currently connected WebSocket clients.",
	})

	wsBroadcasts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ws_broadcasts_total",
		Help: "Total number of board snapshots broadcast to WebSocket rooms.",
	})
)

// metricsMiddleware records request count, latency and in-flight gauge for every
// matched route. It labels by the route template (c.FullPath(), e.g.
// "/api/boards/:id") rather than the raw URL to keep label cardinality bounded.
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		// Skip unmatched routes (404s) and the scrape endpoint itself.
		if path == "" || path == "/metrics" {
			c.Next()
			return
		}

		httpInFlight.Inc()
		start := time.Now()
		c.Next()
		httpInFlight.Dec()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		httpRequests.WithLabelValues(method, path, status).Inc()
		httpDuration.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	}
}
