package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestMaskedRequestPathMasksSensitiveQueryValues(t *testing.T) {
	got := maskedRequestPath("/admin/user/pre_authentication", "user_id=1&pwd=secret&token=abc123&nickname=tester")
	if strings.Contains(got, "secret") {
		t.Fatalf("maskedRequestPath() leaked pwd: %s", got)
	}
	if strings.Contains(got, "abc123") {
		t.Fatalf("maskedRequestPath() leaked token: %s", got)
	}
	if !strings.Contains(got, "nickname=tester") {
		t.Fatalf("maskedRequestPath() should keep non-sensitive params: %s", got)
	}
	if !strings.Contains(got, "pwd=%2A%2A%2A") || !strings.Contains(got, "token=%2A%2A%2A") {
		t.Fatalf("maskedRequestPath() should replace sensitive values: %s", got)
	}
}

func TestRequestLogIncludesRecordedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	router := gin.New()
	router.Use(RequestLog(logger))
	router.GET("/boom", func(c *gin.Context) {
		_ = c.Error(errors.New("jw service status 500: upstream exploded"))
		c.Status(http.StatusInternalServerError)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("log entry count = %d, want 1", len(entries))
	}
	if entries[0].Level != zap.ErrorLevel {
		t.Fatalf("log level = %s, want %s", entries[0].Level, zap.ErrorLevel)
	}

	loggedError, ok := entries[0].ContextMap()["error"].(string)
	if !ok {
		t.Fatalf("expected error field in log context: %#v", entries[0].ContextMap())
	}
	if !strings.Contains(loggedError, "upstream exploded") {
		t.Fatalf("logged error = %q, want to contain upstream detail", loggedError)
	}
}
