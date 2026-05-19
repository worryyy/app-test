package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddlewareUsesFullPathAndBusinessCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New(DependencyChecks{})
	engine := gin.New()
	engine.Use(m.Middleware())
	engine.GET("/api/topic/:topic_id", func(c *gin.Context) {
		SetBusinessCode(c, http.StatusOK)
		SetHTTPStatus(c, http.StatusOK)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/topic/123", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	body := gather(t, m)
	want := `ecampus_http_requests_total{business_code="200",http_status="200",method="GET",route="/api/topic/:topic_id",service="ecampus"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics missing templated route label:\n%s", body)
	}
	if strings.Contains(body, "/api/topic/123") {
		t.Fatalf("metrics leaked raw path:\n%s", body)
	}
}

func TestMiddlewareUsesUnknownBusinessCodeWhenNoBusinessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New(DependencyChecks{})
	engine := gin.New()
	engine.Use(m.Middleware())
	engine.GET("/plain", func(c *gin.Context) {
		c.String(http.StatusNoContent, "")
	})

	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	body := gather(t, m)
	want := `ecampus_http_requests_total{business_code="unknown",http_status="204",method="GET",route="/plain",service="ecampus"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics missing unknown business code:\n%s", body)
	}
	if !strings.Contains(body, `ecampus_business_responses_total{business_code="unknown",route="/plain",service="ecampus"} 1`) {
		t.Fatalf("business response metric missing unknown business code:\n%s", body)
	}
}

func TestMiddlewareUsesUnknownRouteWhenFullPathIsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New(DependencyChecks{})
	engine := gin.New()
	engine.Use(m.Middleware())

	req := httptest.NewRequest(http.MethodGet, "/missing/123", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	body := gather(t, m)
	want := `ecampus_http_requests_total{business_code="unknown",http_status="404",method="GET",route="unknown_route",service="ecampus"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics missing unknown route:\n%s", body)
	}
	if strings.Contains(body, "/missing/123") {
		t.Fatalf("metrics leaked raw missing path:\n%s", body)
	}
}

func TestMetricsEndpointIsNotRecorded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New(DependencyChecks{})
	engine := gin.New()
	engine.Use(m.Middleware())
	engine.GET("/metrics", m.Handler())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, "ecampus_http_requests_total") {
		t.Fatalf("/metrics request should not be recorded:\n%s", body)
	}
}

func TestRefreshDependenciesRecordsUpAndDown(t *testing.T) {
	m := New(DependencyChecks{
		MySQL: func(context.Context) error {
			return nil
		},
		MongoDB: func(context.Context) error {
			return errors.New("down")
		},
		Redis: func(context.Context) error {
			return nil
		},
		RabbitMQ: func(context.Context) error {
			return errors.New("closed")
		},
	})

	m.RefreshDependencies(context.Background())

	body := gather(t, m)
	assertContains(t, body, `ecampus_dependency_up{dependency="mysql",service="ecampus"} 1`)
	assertContains(t, body, `ecampus_dependency_up{dependency="mongodb",service="ecampus"} 0`)
	assertContains(t, body, `ecampus_dependency_up{dependency="redis",service="ecampus"} 1`)
	assertContains(t, body, `ecampus_dependency_up{dependency="rabbitmq",service="ecampus"} 0`)
}

func gather(t *testing.T, m *Metrics) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	engine := gin.New()
	engine.GET("/metrics", m.Handler())
	engine.ServeHTTP(recorder, req)
	return recorder.Body.String()
}

func assertContains(t *testing.T, body, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("metrics missing %q:\n%s", want, body)
	}
}
