package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const unmatchedRoute = "unmatched"

// HTTPMetrics owns the bounded-cardinality metrics used by rollout analysis.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	handler  http.Handler
}

func NewHTTPMetrics(registerer prometheus.Registerer, gatherer prometheus.Gatherer) *HTTPMetrics {
	m := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "ecampus",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests handled by the service.",
		}, []string{"method", "route", "status_code", "outcome"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "ecampus",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"method", "route"}),
		handler: promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}),
	}
	registerer.MustRegister(m.requests, m.duration)
	return m
}

var DefaultHTTPMetrics = NewHTTPMetrics(prometheus.DefaultRegisterer, prometheus.DefaultGatherer)

func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		started := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = unmatchedRoute
		}
		status := c.Writer.Status()
		outcome := "failure"
		if status >= http.StatusOK && status < http.StatusBadRequest {
			outcome = "success"
		}

		method := c.Request.Method
		m.requests.WithLabelValues(method, route, strconv.Itoa(status), outcome).Inc()
		m.duration.WithLabelValues(method, route).Observe(time.Since(started).Seconds())
	}
}

func (m *HTTPMetrics) Handler() http.Handler {
	return m.handler
}
