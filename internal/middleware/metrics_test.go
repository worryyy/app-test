package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newMetricsTestEngine(t *testing.T) (*gin.Engine, *HTTPMetrics) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	registry := prometheus.NewRegistry()
	metrics := NewHTTPMetrics(registry, registry)
	engine := gin.New()
	engine.Use(metrics.Middleware())
	engine.Use(gin.Recovery())
	engine.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.GET("/metrics", gin.WrapH(metrics.Handler()))
	engine.GET("/api/topic/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.GET("/bad-request", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
	engine.GET("/failure", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })
	engine.GET("/panic", func(c *gin.Context) { panic("test panic") })
	return engine, metrics
}

func performRequest(engine http.Handler, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	engine.ServeHTTP(response, request)
	return response
}

func TestHTTPMetricsRecordsFinalStatusAndNormalizedRoute(t *testing.T) {
	engine, metrics := newMetricsTestEngine(t)

	tests := []struct {
		path    string
		status  int
		route   string
		outcome string
	}{
		{path: "/api/topic/42", status: http.StatusOK, route: "/api/topic/:id", outcome: "success"},
		{path: "/bad-request", status: http.StatusBadRequest, route: "/bad-request", outcome: "failure"},
		{path: "/failure", status: http.StatusInternalServerError, route: "/failure", outcome: "failure"},
		{path: "/panic", status: http.StatusInternalServerError, route: "/panic", outcome: "failure"},
	}

	for _, tt := range tests {
		response := performRequest(engine, tt.path)
		if response.Code != tt.status {
			t.Fatalf("GET %s returned %d, want %d", tt.path, response.Code, tt.status)
		}
		if got := testutil.ToFloat64(metrics.requests.WithLabelValues(http.MethodGet, tt.route, strconv.Itoa(tt.status), tt.outcome)); got != 1 {
			t.Fatalf("request metric for %s = %v, want 1", tt.path, got)
		}
	}

	metricsBody := performRequest(engine, "/metrics").Body.String()
	if strings.Contains(metricsBody, `route="/api/topic/42"`) {
		t.Fatal("metrics include a concrete route parameter")
	}
	if !strings.Contains(metricsBody, `route="/api/topic/:id"`) {
		t.Fatal("metrics do not include the normalized Gin route")
	}
	if !strings.Contains(metricsBody, "ecampus_http_request_duration_seconds_count") {
		t.Fatal("histogram was not exposed")
	}
}

func TestHTTPMetricsExcludesHealthAndMetrics(t *testing.T) {
	engine, metrics := newMetricsTestEngine(t)
	performRequest(engine, "/health")
	first := performRequest(engine, "/metrics").Body.String()
	second := performRequest(engine, "/metrics").Body.String()

	if first != second {
		t.Fatal("scraping /metrics changed the exported application metrics")
	}
	if strings.Contains(second, `route="/health"`) || strings.Contains(second, `route="/metrics"`) {
		t.Fatal("health or metrics endpoint was included in rollout metrics")
	}
	if got := testutil.CollectAndCount(metrics.requests); got != 0 {
		t.Fatalf("excluded endpoints created %d request series, want 0", got)
	}
}
