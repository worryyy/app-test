package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	ServiceEcampus      = "ecampus"
	UnknownRoute        = "unknown_route"
	UnknownBusinessCode = "unknown"
)

type Metrics struct {
	service           string
	dependencies      DependencyChecks
	registry          *prometheus.Registry
	httpRequests      *prometheus.CounterVec
	httpDuration      *prometheus.HistogramVec
	businessResponses *prometheus.CounterVec
	dependencyUp      *prometheus.GaugeVec
}

func New(dependencies DependencyChecks) *Metrics {
	httpRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ecampus",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests handled by the service.",
		},
		[]string{"service", "method", "route", "http_status", "business_code"},
	)
	httpDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ecampus",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"service", "method", "route", "http_status", "business_code"},
	)
	businessResponses := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ecampus",
			Name:      "business_responses_total",
			Help:      "Total number of business responses by business code.",
		},
		[]string{"service", "route", "business_code"},
	)
	dependencyUp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ecampus",
			Name:      "dependency_up",
			Help:      "Dependency health status where 1 is up and 0 is down.",
		},
		[]string{"service", "dependency"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(httpRequests, httpDuration, businessResponses, dependencyUp)

	return &Metrics{
		service:           ServiceEcampus,
		dependencies:      dependencies,
		registry:          registry,
		httpRequests:      httpRequests,
		httpDuration:      httpDuration,
		businessResponses: businessResponses,
		dependencyUp:      dependencyUp,
	}
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil {
			c.Next()
			return
		}
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = UnknownRoute
		}
		businessCode, ok := BusinessCode(c)
		if !ok {
			businessCode = UnknownBusinessCode
		}
		httpStatus := HTTPStatus(c, c.Writer.Status())

		labels := prometheus.Labels{
			"service":       m.service,
			"method":        c.Request.Method,
			"route":         route,
			"http_status":   httpStatus,
			"business_code": businessCode,
		}
		m.httpRequests.With(labels).Inc()
		m.httpDuration.With(labels).Observe(time.Since(start).Seconds())
		m.businessResponses.With(prometheus.Labels{
			"service":       m.service,
			"route":         route,
			"business_code": businessCode,
		}).Inc()
	}
}

func (m *Metrics) Handler() gin.HandlerFunc {
	if m == nil {
		return func(c *gin.Context) {
			c.Status(http.StatusNotFound)
		}
	}
	handler := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		m.RefreshDependencies(ctx)
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

func (m *Metrics) RefreshDependencies(ctx context.Context) {
	for _, dep := range m.dependencies.all() {
		up := 0.0
		if dep.check != nil && dep.check(ctx) == nil {
			up = 1
		}
		m.dependencyUp.WithLabelValues(m.service, dep.name).Set(up)
	}
}
