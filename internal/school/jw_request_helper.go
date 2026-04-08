package school

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/encrypt"
)

const (
	defaultJWAuthBaseURL  = "https://auth.sztu.edu.cn/idp"
	defaultJWJSXSDBaseURL = "https://jwxt.sztu.edu.cn/jsxsd"
	defaultJWAuthClass    = "urn_oasis_names_tc_SAML_2.0_ac_classes_BAMUsernamePassword"
	defaultJWUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	defaultJWFailureMsg   = "登陆失败，请检查账号密码或者登陆状态是否异常"
	defaultJWSuccessMsg   = "success"
	jwLoginPasswordDESKey = "PassB01I"
)

type JWRequestHelper struct {
	cfg          *config.Config
	logger       *zap.Logger
	userAgent    string
	authBaseURL  string
	authChainURL string
	authAjaxURL  string
	authSubmit   string
	jsxsdBaseURL string
}

type jwSession struct {
	helper *JWRequestHelper
	client *http.Client
}

type jwLoginMeta struct {
	Name  string
	Major string
}

type jwAuthCenterResp struct {
	View                   string `json:"view"`
	LoginFailed            any    `json:"loginFailed"`
	ShowViewExcepMsg       string `json:"showViewExcepMsg"`
	AuthenticationErrorTip string `json:"authenticationErrorTip"`
	AuthenticationInfoTip  string `json:"authenticationInfoTip"`
	ErrorMsg               string `json:"errorMsg"`
	Msg                    string `json:"msg"`
}

type jwPageRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Form    url.Values
	Referer string
}

func NewJWRequestHelper(cfg *config.Config, logger *zap.Logger) *JWRequestHelper {
	if logger == nil {
		logger = zap.NewNop()
	}

	authBaseURL := strings.TrimRight(defaultJWAuthBaseURL, "/")
	jsxsdBaseURL := strings.TrimRight(defaultJWJSXSDBaseURL, "/")
	if cfg != nil {
		if v := strings.TrimSpace(cfg.JW.AuthURL); v != "" {
			authBaseURL = strings.TrimRight(v, "/")
		}
		if v := strings.TrimSpace(cfg.JW.JSXSDURL); v != "" {
			jsxsdBaseURL = strings.TrimRight(v, "/")
		}
	}

	return &JWRequestHelper{
		cfg:          cfg,
		logger:       logger,
		userAgent:    defaultJWUserAgent,
		authBaseURL:  authBaseURL,
		authChainURL: authBaseURL + "/authcenter/ActionAuthChain?entityId=jiaowu",
		authAjaxURL:  authBaseURL + "/authcenter/ActionAuthChain",
		authSubmit:   authBaseURL + "/AuthnEngine?currentAuth=" + url.QueryEscape(defaultJWAuthClass),
		jsxsdBaseURL: jsxsdBaseURL,
	}
}

func (h *JWRequestHelper) CheckLogin(ctx context.Context, schoolID, password string) (*JWCommonResp, error) {
	_, meta, failedResp, err := h.login(ctx, schoolID, password)
	if failedResp != nil || err != nil {
		return failedResp, err
	}

	return &JWCommonResp{
		Code:    http.StatusOK,
		Message: defaultJWSuccessMsg,
		Data: map[string]any{
			"is_login": true,
			"name":     meta.Name,
			"major":    meta.Major,
		},
	}, nil
}

func (h *JWRequestHelper) GetCourseByWeeks(ctx context.Context, startDate string, week int, req JWGetCourseReq) (*JWCommonResp, error) {
	session, _, failedResp, err := h.login(ctx, req.SchoolID, req.Password)
	if failedResp != nil || err != nil {
		return failedResp, err
	}

	targetDate := computeCourseDate(startDate, week)
	body, err := session.fetchCourseWeekHTML(ctx, targetDate)
	if err != nil {
		return nil, err
	}

	return &JWCommonResp{
		Code:    http.StatusOK,
		Message: defaultJWSuccessMsg,
		Data:    parseWeeklyCourseHTML(body, req.Term, targetDate, week),
	}, nil
}

func (h *JWRequestHelper) GetExam(ctx context.Context, req JWGetExamReq) (*JWCommonResp, error) {
	session, _, failedResp, err := h.login(ctx, req.SchoolID, req.Password)
	if failedResp != nil || err != nil {
		return failedResp, err
	}

	body, err := session.fetchExamHTML(ctx, req.XNXQID)
	if err != nil {
		return nil, err
	}

	return &JWCommonResp{
		Code:    http.StatusOK,
		Message: defaultJWSuccessMsg,
		Data:    parseHTMLTableRows(body),
	}, nil
}

func (h *JWRequestHelper) GetExamScore(ctx context.Context, req JWGetExamScoreReq) (*JWCommonResp, error) {
	session, _, failedResp, err := h.login(ctx, req.SchoolID, req.Password)
	if failedResp != nil || err != nil {
		return failedResp, err
	}

	body, err := session.fetchExamScoreHTML(ctx, req.SS)
	if err != nil {
		return nil, err
	}

	return &JWCommonResp{
		Code:    http.StatusOK,
		Message: defaultJWSuccessMsg,
		Data:    parseHTMLTableRows(body),
	}, nil
}

func (h *JWRequestHelper) login(ctx context.Context, schoolID, password string) (*jwSession, *jwLoginMeta, *JWCommonResp, error) {
	session, err := h.newSession()
	if err != nil {
		return nil, nil, nil, err
	}
	if _, err := session.get(ctx, h.authChainURL, h.authChainURL); err != nil {
		return nil, nil, nil, fmt.Errorf("init jw auth session: %w", err)
	}

	form := url.Values{
		"j_username":      {strings.TrimSpace(schoolID)},
		"j_password":      {h.encryptByDES(password)},
		"op":              {"login"},
		"spAuthChainCode": {""},
	}

	preflightBody, err := session.postForm(ctx, h.authAjaxURL, form, true, h.authChainURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("preflight jw auth: %w", err)
	}
	if authResp, ok := decodeJWAuthCenterResp(preflightBody); ok && authCenterFailed(authResp.LoginFailed) {
		return nil, nil, buildJWFailureResp(authResp), nil
	}

	if _, err := session.postForm(ctx, h.authSubmit, form, false, h.authChainURL); err != nil {
		return nil, nil, nil, fmt.Errorf("submit jw auth: %w", err)
	}

	mainBody, err := session.get(ctx, session.helper.jsxsdBaseURL+"/framework/xsMain.htmlx", session.helper.authChainURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open jw main page: %w", err)
	}
	if looksLikeJWLoginPage(mainBody) {
		return nil, nil, &JWCommonResp{Code: http.StatusBadRequest, Message: defaultJWFailureMsg}, nil
	}

	meta := &jwLoginMeta{}
	if name, major, ok := parseProfileMeta(mainBody); ok {
		meta.Name = name
		meta.Major = major
	}
	if meta.Name == "" || meta.Major == "" {
		if profileBody, profileErr := session.fetchProfileHTML(ctx); profileErr == nil {
			if name, major, ok := parseProfileMeta(profileBody); ok {
				if meta.Name == "" {
					meta.Name = name
				}
				if meta.Major == "" {
					meta.Major = major
				}
			}
		}
	}
	return session, meta, nil, nil
}

func (h *JWRequestHelper) newSession() (*jwSession, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create jw cookie jar: %w", err)
	}

	return &jwSession{
		helper: h,
		client: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
	}, nil
}

func (s *jwSession) fetchProfileHTML(ctx context.Context) (string, error) {
	candidates := []jwPageRequest{
		{Method: http.MethodGet, Path: "/xsgrxx/xsxx", Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx"},
		{Method: http.MethodGet, Path: "/grxx/xsxx", Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx"},
	}
	return s.fetchFirstPage(ctx, candidates)
}

func (s *jwSession) fetchCourseWeekHTML(ctx context.Context, targetDate string) (string, error) {
	return s.fetchFirstPage(ctx, []jwPageRequest{
		{
			Method:  http.MethodPost,
			Path:    "/framework/main_index_loadkb.jsp",
			Query:   url.Values{"rq": {targetDate}},
			Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx",
		},
	})
}

func (s *jwSession) fetchExamHTML(ctx context.Context, term string) (string, error) {
	return s.fetchFirstPage(ctx, examRequests(s.helper.jsxsdBaseURL, term))
}

func (s *jwSession) fetchExamScoreHTML(ctx context.Context, term string) (string, error) {
	form := url.Values{"kcxz": {""}, "kcmc": {""}, "xsfs": {"all"}}
	if strings.TrimSpace(term) != "" {
		form.Set("kksj", term)
	}
	return s.fetchFirstPage(ctx, []jwPageRequest{
		{
			Method:  http.MethodPost,
			Path:    "/kscj/cjcx_list",
			Form:    form,
			Referer: s.helper.jsxsdBaseURL + "/kscj/cjcx_query",
		},
	})
}

func (s *jwSession) fetchFirstPage(ctx context.Context, candidates []jwPageRequest) (string, error) {
	var lastErr error
	for _, candidate := range candidates {
		body, err := s.requestPage(ctx, candidate)
		if err != nil {
			lastErr = err
			continue
		}
		if looksLikeJWLoginPage(body) {
			lastErr = fmt.Errorf("jw session expired while requesting %s", candidate.Path)
			continue
		}
		if hasUsefulHTML(body) {
			return body, nil
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("jw page fetch failed")
}

func (s *jwSession) requestPage(ctx context.Context, req jwPageRequest) (string, error) {
	endpoint := strings.TrimRight(s.helper.jsxsdBaseURL, "/") + req.Path
	if len(req.Query) > 0 {
		endpoint += "?" + req.Query.Encode()
	}
	switch req.Method {
	case http.MethodPost:
		return s.postForm(ctx, endpoint, req.Form, false, req.Referer)
	default:
		return s.get(ctx, endpoint, req.Referer)
	}
}

func (s *jwSession) get(ctx context.Context, endpoint string, referer string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	s.applyHeaders(req, false, referer)
	return s.do(req)
}

func (s *jwSession) postForm(ctx context.Context, endpoint string, form url.Values, ajax bool, referer string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	s.applyHeaders(req, ajax, referer)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	return s.do(req)
}

func (s *jwSession) applyHeaders(req *http.Request, ajax bool, referer string) {
	req.Header.Set("User-Agent", s.helper.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if ajax {
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
		if u, err := url.Parse(referer); err == nil {
			req.Header.Set("Origin", u.Scheme+"://"+u.Host)
		}
	}
}

func (s *jwSession) do(req *http.Request) (string, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return string(raw), nil
}

func (h *JWRequestHelper) encryptByDES(password string) string {
	out, err := encrypt.DESECBEncrypt([]byte(password), []byte(jwLoginPasswordDESKey))
	if err != nil {
		h.logger.Warn("encrypt jw password failed", zap.Error(err))
		return ""
	}
	return base64.StdEncoding.EncodeToString(out)
}

func decodeJWAuthCenterResp(raw string) (*jwAuthCenterResp, bool) {
	var out jwAuthCenterResp
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	if out.View == "" && out.LoginFailed == nil && out.ShowViewExcepMsg == "" && out.AuthenticationErrorTip == "" && out.Msg == "" {
		return nil, false
	}
	return &out, true
}

func authCenterFailed(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true")
	default:
		return false
	}
}

func buildJWFailureResp(authResp *jwAuthCenterResp) *JWCommonResp {
	msg := defaultJWFailureMsg
	if authResp != nil {
		msg = firstNonEmpty(
			strings.TrimSpace(authResp.AuthenticationErrorTip),
			strings.TrimSpace(authResp.ShowViewExcepMsg),
			strings.TrimSpace(authResp.ErrorMsg),
			strings.TrimSpace(authResp.Msg),
			msg,
		)
	}
	return &JWCommonResp{Code: http.StatusBadRequest, Message: msg}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func computeCourseDate(startDate string, week int) string {
	base, err := time.Parse("2006-01-02", strings.TrimSpace(startDate))
	if err != nil {
		return strings.TrimSpace(startDate)
	}
	if week > 1 {
		base = base.AddDate(0, 0, (week-1)*7)
	}
	return base.Format("2006-01-02")
}

func examRequests(baseURL string, term string) []jwPageRequest {
	values := []jwPageRequest{
		{Method: http.MethodGet, Path: "/xsks/xsksap_list", Referer: strings.TrimRight(baseURL, "/") + "/framework/xsMain.htmlx"},
		{Method: http.MethodGet, Path: "/xsksap/xsksap_list", Referer: strings.TrimRight(baseURL, "/") + "/framework/xsMain.htmlx"},
	}
	if strings.TrimSpace(term) == "" {
		return values
	}

	queryNames := []string{"xnxq01id", "xnxqid", "kksj"}
	var out []jwPageRequest
	for _, req := range values {
		out = append(out, req)
		for _, name := range queryNames {
			query := url.Values{name: {term}}
			req.Query = query
			out = append(out, req)
			req.Method = http.MethodPost
			req.Query = nil
			req.Form = query
			out = append(out, req)
		}
	}
	return out
}
