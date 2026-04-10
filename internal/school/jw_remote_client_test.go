package school

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

func TestJWClientCheckLoginUsesRemoteService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/check_login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-api-key" {
			t.Fatalf("unexpected api key: %q", got)
		}

		var req JWLoginReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.SchoolID != "2023001" || req.Password != "jw-pass" {
			t.Fatalf("unexpected request body: %#v", req)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"is_login":true,"name":"张三","major":"计科"}}`))
	}))
	defer server.Close()

	client := NewJWClient(&config.Config{
		JW: config.JWConfig{
			BaseURL: server.URL,
			APIKey:  "test-api-key",
		},
	}, zap.NewNop())

	resp, err := client.CheckLogin(context.Background(), "2023001", "jw-pass")
	if err != nil {
		t.Fatalf("CheckLogin returned error: %v", err)
	}
	if resp == nil || resp.Code != http.StatusOK || resp.Message != "success" {
		t.Fatalf("unexpected response: %#v", resp)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if data["is_login"] != true || data["name"] != "张三" {
		t.Fatalf("unexpected response data: %#v", data)
	}
}

func TestJWClientGetCourseByWeeksUsesQueryAndJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/get_course_by_weeks" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("date") != "2025-09-01" {
			t.Fatalf("unexpected date query: %s", query.Get("date"))
		}
		if query.Get("weeks") != "2" {
			t.Fatalf("unexpected weeks query: %s", query.Get("weeks"))
		}

		var req JWGetCourseReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Term != "2025-2026-1" || req.SchoolID != "2023001" || req.Password != "jw-pass" {
			t.Fatalf("unexpected request body: %#v", req)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":[{"course":"高数","week":2}]}`))
	}))
	defer server.Close()

	client := NewJWClient(&config.Config{
		JW: config.JWConfig{BaseURL: server.URL},
	}, zap.NewNop())

	resp, err := client.GetCourseByWeeks(context.Background(), "2025-09-01", 2, JWGetCourseReq{
		Term:     "2025-2026-1",
		SchoolID: "2023001",
		Password: "jw-pass",
	})
	if err != nil {
		t.Fatalf("GetCourseByWeeks returned error: %v", err)
	}
	if resp == nil || resp.Code != http.StatusOK {
		t.Fatalf("unexpected response: %#v", resp)
	}

	items, ok := resp.Data.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected response data: %#v", resp.Data)
	}
}

func TestJWClientReturnsBusinessResponseOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get_exam" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":400,"message":"bad credentials","data":null}`))
	}))
	defer server.Close()

	client := NewJWClient(&config.Config{
		JW: config.JWConfig{BaseURL: server.URL},
	}, zap.NewNop())

	resp, err := client.GetExam(context.Background(), JWGetExamReq{
		SchoolID: "2023001",
		Password: "bad-pass",
		XNXQID:   "2025-2026-1",
	})
	if err != nil {
		t.Fatalf("GetExam returned error: %v", err)
	}
	if resp == nil || resp.Code != http.StatusBadRequest || resp.Message != "bad credentials" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestJWRemoteClientResolveURLKeepsBasePath(t *testing.T) {
	client := &jwRemoteClient{baseURL: "https://example.com/sztu_jw"}
	endpoint, err := client.resolveURL("/get_exam_score", url.Values{"ss": {"2025-2026-1"}})
	if err != nil {
		t.Fatalf("resolveURL returned error: %v", err)
	}
	want := "https://example.com/sztu_jw/get_exam_score?ss=2025-2026-1"
	if endpoint != want {
		t.Fatalf("unexpected endpoint: got %s want %s", endpoint, want)
	}
}

func TestJWClientUnreadableHTTPErrorIncludesResponsePreview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html><body>jw upstream exploded</body></html>"))
	}))
	defer server.Close()

	client := NewJWClient(&config.Config{
		JW: config.JWConfig{BaseURL: server.URL},
	}, zap.NewNop())

	_, err := client.GetExam(context.Background(), JWGetExamReq{
		SchoolID: "2023001",
		Password: "bad-pass",
		XNXQID:   "2025-2026-1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "jw service status 500") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "jw upstream exploded") {
		t.Fatalf("error should include response preview: %v", err)
	}
}
