package responses

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	if body.Message != "login ok" {
		t.Fatalf("message = %q, want login ok", body.Message)
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
	if body.Code != CodeParamErr {
		t.Fatalf("code = %d, want %d", body.Code, CodeParamErr)
	}
}
