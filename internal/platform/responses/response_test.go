package responses

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/metrics"
)

func TestRespMessageData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Success.RespMessageData(ctx, "login ok", gin.H{"id": 1})

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Code != CodeSuccess {
		t.Fatalf("code = %d, want %d", body.Code, CodeSuccess)
	}
	if body.HTTPStatus != HTTPStatusOK {
		t.Fatalf("httpstatus = %d, want %d", body.HTTPStatus, HTTPStatusOK)
	}
	if body.Code != recorder.Code {
		t.Fatalf("body code = %d, want status %d", body.Code, recorder.Code)
	}
	if body.Msg != "login ok" {
		t.Fatalf("msg = %q, want login ok", body.Msg)
	}
}

func TestRespUsesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("request_id", "req-123")

	ParamErr.RespMessage(ctx, "username required")

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.RequestID != "req-123" {
		t.Fatalf("requestId = %q, want req-123", body.RequestID)
	}
	if recorder.Code != CodeFail {
		t.Fatalf("status = %d, want %d", recorder.Code, CodeFail)
	}
	if body.Code != CodeFail {
		t.Fatalf("code = %d, want %d", body.Code, CodeFail)
	}
	if body.HTTPStatus != HTTPStatusBadRequest {
		t.Fatalf("httpstatus = %d, want %d", body.HTTPStatus, HTTPStatusBadRequest)
	}
	if body.Code != recorder.Code {
		t.Fatalf("body code = %d, want status %d", body.Code, recorder.Code)
	}
}

func TestErrorHTTPStatusUsesDetailedStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	New(false, http.StatusUnauthorized, "token invalid").Resp(ctx)

	if recorder.Code != CodeFail {
		t.Fatalf("status = %d, want %d", recorder.Code, CodeFail)
	}

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Code != CodeFail {
		t.Fatalf("code = %d, want %d", body.Code, CodeFail)
	}
	if body.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("httpstatus = %d, want %d", body.HTTPStatus, http.StatusUnauthorized)
	}
}

func TestFailRecordsErrorOnContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Fail(ctx, errors.New("upstream exploded"))

	if len(ctx.Errors) != 1 {
		t.Fatalf("error count = %d, want 1", len(ctx.Errors))
	}
	if got := ctx.Errors.Last().Err.Error(); got != "upstream exploded" {
		t.Fatalf("last error = %q, want upstream exploded", got)
	}
}

func TestResponseWritesBusinessCodeForMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Success.Resp(ctx)

	code, ok := metrics.BusinessCode(ctx)
	if !ok {
		t.Fatalf("business code missing")
	}
	if code != "200" {
		t.Fatalf("business code = %q, want 200", code)
	}
	if got := metrics.HTTPStatus(ctx, 0); got != "200" {
		t.Fatalf("http status = %q, want 200", got)
	}
}

func TestFailWritesBusinessCodeForMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Fail(ctx, errors.New("upstream exploded"))

	code, ok := metrics.BusinessCode(ctx)
	if !ok {
		t.Fatalf("business code missing")
	}
	if code != "400" {
		t.Fatalf("business code = %q, want 400", code)
	}
	if got := metrics.HTTPStatus(ctx, 0); got != "500" {
		t.Fatalf("http status = %q, want 500", got)
	}
}
