package school

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.uber.org/zap"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/encrypt"
)

func TestJWRequestHelperCheckLoginSuccess(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			if got := r.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
				t.Fatalf("unexpected X-Requested-With: %q", got)
			}
			values := mustParseForm(t, r)
			if values.Get("j_username") != "2023001" {
				t.Fatalf("unexpected username: %q", values.Get("j_username"))
			}
			if values.Get("j_password") != mustEncryptDES(t, "jw-pass") {
				t.Fatalf("unexpected encrypted password: %q", values.Get("j_password"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.htmlx", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			_, _ = w.Write([]byte(`<table><tr><td>姓名</td><td>张三</td></tr><tr><td>专业</td><td>计算机科学与技术</td></tr></table>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.CheckLogin(context.Background(), "2023001", "jw-pass")
	if err != nil {
		t.Fatalf("CheckLogin returned error: %v", err)
	}
	if resp == nil || resp.Code != http.StatusOK || resp.Message != defaultJWSuccessMsg {
		t.Fatalf("unexpected resp: %#v", resp)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type: %T", resp.Data)
	}
	if data["is_login"] != true || data["name"] != "张三" || data["major"] != "计算机科学与技术" {
		t.Fatalf("unexpected login data: %#v", data)
	}
}

func TestJWRequestHelperCheckLoginFailure(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":true,"showViewExcepMsg":"账号或密码错误"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.CheckLogin(context.Background(), "2023001", "bad-pass")
	if err != nil {
		t.Fatalf("CheckLogin returned error: %v", err)
	}
	if resp == nil || resp.Code != http.StatusBadRequest || resp.Message != "账号或密码错误" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
}

func TestJWRequestHelperGetCourseByWeeksParsesWeeklyBlocks(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.htmlx", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			_, _ = w.Write([]byte(`<table><tr><td>姓名</td><td>张三</td></tr></table>`))
		case r.Method == http.MethodPost && r.URL.Path == "/jsxsd/framework/main_index_loadkb.jsp":
			if got := r.URL.Query().Get("rq"); got != "2025-09-08" {
				t.Fatalf("unexpected rq query: %q", got)
			}
			_, _ = w.Write([]byte(`<div><p>上课时间：第2周 星期一 [01-02]节<br/>课程名称：高等数学<br/>上课地点：A101<br/>教师：张老师</p></div>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.GetCourseByWeeks(context.Background(), "2025-09-01", 2, JWGetCourseReq{
		Term:     "2025-2026-1",
		SchoolID: "2023001",
		Password: "jw-pass",
	})
	if err != nil {
		t.Fatalf("GetCourseByWeeks returned error: %v", err)
	}

	items, ok := resp.Data.([]map[string]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected course payload: %#v", resp.Data)
	}
	if items[0]["course"] != "高等数学" || items[0]["location"] != "A101" || items[0]["section"] != "01-02" {
		t.Fatalf("unexpected course row: %#v", items[0])
	}
}

func TestJWRequestHelperGetExamScoreParsesTable(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.htmlx", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			_, _ = w.Write([]byte(`<table><tr><td>姓名</td><td>张三</td></tr></table>`))
		case r.Method == http.MethodPost && r.URL.Path == "/jsxsd/kscj/cjcx_list":
			values := mustParseForm(t, r)
			if values.Get("kksj") != "2025-2026-1" {
				t.Fatalf("unexpected kksj: %q", values.Get("kksj"))
			}
			_, _ = w.Write([]byte(`<table><tr><th>课程名称</th><th>成绩</th></tr><tr><td>高等数学</td><td>95</td></tr></table>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.GetExamScore(context.Background(), JWGetExamScoreReq{
		SchoolID: "2023001",
		Password: "jw-pass",
		SS:       "2025-2026-1",
	})
	if err != nil {
		t.Fatalf("GetExamScore returned error: %v", err)
	}

	items, ok := resp.Data.([]map[string]string)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected score payload: %#v", resp.Data)
	}
	if items[0]["课程名称"] != "高等数学" || items[0]["成绩"] != "95" {
		t.Fatalf("unexpected score row: %#v", items[0])
	}
}

func TestJWRequestHelperGetExamUsesCandidatePage(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.htmlx", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			_, _ = w.Write([]byte(`<table><tr><td>姓名</td><td>张三</td></tr></table>`))
		case r.URL.Path == "/jsxsd/xsks/xsksap_list" || r.URL.Path == "/jsxsd/xsksap/xsksap_list":
			_, _ = w.Write([]byte(`<table><tr><th>课程</th><th>时间</th></tr><tr><td>高等数学</td><td>2026-01-08 09:00</td></tr></table>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.GetExam(context.Background(), JWGetExamReq{
		SchoolID: "2023001",
		Password: "jw-pass",
		XNXQID:   "2025-2026-1",
	})
	if err != nil {
		t.Fatalf("GetExam returned error: %v", err)
	}

	items, ok := resp.Data.([]map[string]string)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected exam payload: %#v", resp.Data)
	}
	if items[0]["课程"] != "高等数学" || items[0]["时间"] != "2026-01-08 09:00" {
		t.Fatalf("unexpected exam row: %#v", items[0])
	}
}

func TestJWRequestHelperGetExamScoreRetriesTransientFailure(t *testing.T) {
	scoreAttempts := 0
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.htmlx", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			_, _ = w.Write([]byte(`<table><tr><td>姓名</td><td>张三</td></tr></table>`))
		case r.Method == http.MethodPost && r.URL.Path == "/jsxsd/kscj/cjcx_list":
			scoreAttempts++
			if scoreAttempts == 1 {
				http.Error(w, "temporary upstream error", http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`<table><tr><th>课程名称</th><th>成绩</th></tr><tr><td>高等数学</td><td>95</td></tr></table>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.GetExamScore(context.Background(), JWGetExamScoreReq{
		SchoolID: "2023001",
		Password: "jw-pass",
		SS:       "2025-2026-1",
	})
	if err != nil {
		t.Fatalf("GetExamScore returned error: %v", err)
	}

	items, ok := resp.Data.([]map[string]string)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected score payload: %#v", resp.Data)
	}
	if scoreAttempts != 2 {
		t.Fatalf("expected 2 score attempts, got %d", scoreAttempts)
	}
}

func TestJWRequestHelperCheckLoginFallsBackToAlternateMainPage(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.jsp", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.jsp":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			_, _ = w.Write([]byte(`<table><tr><td>姓名</td><td>张三</td></tr><tr><td>专业</td><td>计算机科学与技术</td></tr></table>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.CheckLogin(context.Background(), "2023001", "jw-pass")
	if err != nil {
		t.Fatalf("CheckLogin returned error: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type: %T", resp.Data)
	}
	if data["name"] != "张三" || data["major"] != "计算机科学与技术" {
		t.Fatalf("unexpected login data: %#v", data)
	}
}

func TestJWRequestHelperCheckLoginDecodesGBKProfilePage(t *testing.T) {
	server := newJWTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			_, _ = w.Write([]byte(`<html>login</html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/authcenter/ActionAuthChain":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"loginFailed":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/idp/AuthnEngine":
			http.Redirect(w, r, "/jsxsd/framework/xsMain.htmlx", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/framework/xsMain.htmlx":
			_, _ = w.Write([]byte(`<html><body>欢迎您：张三</body></html>`))
		case r.Method == http.MethodGet && r.URL.Path == "/jsxsd/xsgrxx/xsxx":
			w.Header().Set("Content-Type", "text/html; charset=gbk")
			_, _ = w.Write(mustEncodeGBK(t, `<table><tr><td>姓名</td><td>张三</td></tr><tr><td>专业</td><td>计算机科学与技术</td></tr></table>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	defer server.Close()

	helper := newTestJWHelper(server.URL)
	resp, err := helper.CheckLogin(context.Background(), "2023001", "jw-pass")
	if err != nil {
		t.Fatalf("CheckLogin returned error: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type: %T", resp.Data)
	}
	if data["name"] != "张三" || data["major"] != "计算机科学与技术" {
		t.Fatalf("unexpected login data: %#v", data)
	}
}

func newTestJWHelper(serverURL string) *JWRequestHelper {
	return NewJWRequestHelper(&config.Config{
		JW: config.JWConfig{
			AuthURL:  serverURL + "/idp",
			JSXSDURL: serverURL + "/jsxsd",
		},
	}, zap.NewNop())
}

func newJWTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(handler))
}

func mustParseForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("ParseForm returned error: %v", err)
	}
	return r.PostForm
}

func mustEncryptDES(t *testing.T, raw string) string {
	t.Helper()
	out, err := encrypt.DESECBEncrypt([]byte(raw), []byte(jwLoginPasswordDESKey))
	if err != nil {
		t.Fatalf("DESECBEncrypt returned error: %v", err)
	}
	return base64.StdEncoding.EncodeToString(out)
}

func mustEncodeGBK(t *testing.T, raw string) []byte {
	t.Helper()
	out, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(raw))
	if err != nil {
		t.Fatalf("encode gbk returned error: %v", err)
	}
	return out
}
