package ecampus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/agentchat"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/metrics"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

func TestRegisterRoutesExposesMetricsOnlyForUserService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	appMetrics := metrics.New(metrics.DependencyChecks{})

	registerRoutes(engine, zap.NewNop(), nil, nil, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), appMetrics, testUserHandlers())

	routes := routeSet(engine.Routes())
	if !routes[routeKey{method: "GET", path: "/metrics"}] {
		t.Fatalf("missing route GET /metrics")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "ecampus_dependency_up") {
		t.Fatalf("metrics body missing dependency metric:\n%s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "ecampus_http_requests_total") {
		t.Fatalf("/metrics request should not be recorded:\n%s", recorder.Body.String())
	}
}

func TestRegisterRoutesMetricsUsesTemplatedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	appMetrics := metrics.New(metrics.DependencyChecks{})
	engine.Use(appMetrics.Middleware())
	engine.GET("/api/topic/:topic_id", func(c *gin.Context) {
		c.String(http.StatusNoContent, "")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/topic/123", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRecorder := httptest.NewRecorder()
	metricsEngine := gin.New()
	metricsEngine.GET("/metrics", appMetrics.Handler())
	metricsEngine.ServeHTTP(metricsRecorder, metricsReq)
	body := metricsRecorder.Body.String()
	want := `ecampus_http_requests_total{business_code="unknown",http_status="204",method="GET",route="/api/topic/:topic_id",service="ecampus"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics missing templated topic route:\n%s", body)
	}
	if strings.Contains(body, "/api/topic/123") {
		t.Fatalf("metrics leaked raw topic path:\n%s", body)
	}
}

type routeKey struct {
	method string
	path   string
}

func routeSet(routes gin.RoutesInfo) map[routeKey]bool {
	out := make(map[routeKey]bool, len(routes))
	for _, route := range routes {
		out[routeKey{method: route.Method, path: route.Path}] = true
	}
	return out
}

func testUserHandlers() userHandlers {
	return userHandlers{
		User:    user.NewHandler(nil),
		Topic:   topic.NewHandler(nil),
		Comment: comment.NewHandler(nil),
		Theme:   theme.NewHandler(nil),
		File:    file.NewHandler(nil),
		Chat:    chat.NewHandler(nil, nil, nil, nil),
		School:  school.NewHandler(nil),
		Agent:   agentchat.NewHandler(nil, nil, nil),
	}
}
