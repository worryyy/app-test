package school

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/html/charset"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/encrypt"
)

const (
	defaultJWAuthBaseURL    = "https://auth.sztu.edu.cn/idp"
	defaultJWJSXSDBaseURL   = "https://jwxt.sztu.edu.cn/jsxsd"
	defaultJWAuthClass      = "urn_oasis_names_tc_SAML_2.0_ac_classes_BAMUsernamePassword"
	defaultJWUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	defaultJWFailureMsg     = "登陆失败，请检查账号密码或者登陆状态是否异常"
	defaultJWSuccessMsg     = "success"
	jwLoginPasswordDESKey   = "PassB01I"
	defaultJWRequestTimeout = 20 * time.Second
	defaultJWRequestTries   = 2
	defaultJWRetryBackoff   = 200 * time.Millisecond
)

type JWRequestHelper struct {
	cfg             *config.Config
	logger          *zap.Logger
	userAgent       string
	authBaseURL     string
	authChainURL    string
	authAjaxURL     string
	authSubmit      string
	jsxsdBaseURL    string
	requestTimeout  time.Duration
	requestAttempts int
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
	Name    string
	Method  string
	Path    string
	Query   url.Values
	Form    url.Values
	Referer string
}

type jwHTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *jwHTTPStatusError) Error() string {
	if e == nil {
		return ""
	}
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("status %d", e.StatusCode)
	}
	return fmt.Sprintf("status %d: %s", e.StatusCode, body)
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
		cfg:             cfg,
		logger:          logger,
		userAgent:       defaultJWUserAgent,
		authBaseURL:     authBaseURL,
		authChainURL:    authBaseURL + "/authcenter/ActionAuthChain?entityId=jiaowu",
		authAjaxURL:     authBaseURL + "/authcenter/ActionAuthChain",
		authSubmit:      authBaseURL + "/AuthnEngine?currentAuth=" + url.QueryEscape(defaultJWAuthClass),
		jsxsdBaseURL:    jsxsdBaseURL,
		requestTimeout:  defaultJWRequestTimeout,
		requestAttempts: defaultJWRequestTries,
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
	if err := session.initAuthSession(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("init jw auth session: %w", err)
	}

	form, err := h.buildLoginForm(schoolID, password)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("build jw login form: %w", err)
	}

	failedResp, err := session.preflightLogin(ctx, form)
	if failedResp != nil || err != nil {
		return nil, nil, failedResp, err
	}

	if err := session.submitLogin(ctx, form); err != nil {
		return nil, nil, nil, fmt.Errorf("submit jw auth: %w", err)
	}

	mainBody, err := session.openMainPage(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open jw main page: %w", err)
	}
	if looksLikeJWLoginPage(mainBody) {
		return nil, nil, &JWCommonResp{Code: http.StatusBadRequest, Message: defaultJWFailureMsg}, nil
	}

	meta := session.resolveLoginMeta(ctx, mainBody)
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
			Timeout: h.requestTimeout,
		},
	}, nil
}

func (h *JWRequestHelper) buildLoginForm(schoolID, password string) (url.Values, error) {
	encrypted, err := h.encryptByDES(password)
	if err != nil {
		return nil, err
	}

	return url.Values{
		"j_username":      {strings.TrimSpace(schoolID)},
		"j_password":      {encrypted},
		"op":              {"login"},
		"spAuthChainCode": {""},
	}, nil
}

func (s *jwSession) initAuthSession(ctx context.Context) error {
	_, err := s.get(ctx, s.helper.authChainURL, s.helper.authChainURL)
	return err
}

func (s *jwSession) preflightLogin(ctx context.Context, form url.Values) (*JWCommonResp, error) {
	preflightBody, err := s.postForm(ctx, s.helper.authAjaxURL, form, true, s.helper.authChainURL)
	if err != nil {
		return nil, fmt.Errorf("preflight jw auth: %w", err)
	}
	if authResp, ok := decodeJWAuthCenterResp(preflightBody); ok && authCenterFailed(authResp.LoginFailed) {
		return buildJWFailureResp(authResp), nil
	}
	return nil, nil
}

func (s *jwSession) submitLogin(ctx context.Context, form url.Values) error {
	_, err := s.postForm(ctx, s.helper.authSubmit, form, false, s.helper.authChainURL)
	return err
}

func (s *jwSession) openMainPage(ctx context.Context) (string, error) {
	return s.fetchFirstPage(ctx, []jwPageRequest{
		{
			Name:    "main_htmlx",
			Method:  http.MethodGet,
			Path:    "/framework/xsMain.htmlx",
			Referer: s.helper.authChainURL,
		},
		{
			Name:    "main_jsp",
			Method:  http.MethodGet,
			Path:    "/framework/xsMain.jsp",
			Referer: s.helper.authChainURL,
		},
	}, nil)
}

func (s *jwSession) resolveLoginMeta(ctx context.Context, mainBody string) *jwLoginMeta {
	meta := &jwLoginMeta{}
	if name, major, ok := parseProfileMeta(mainBody); ok {
		meta.Name = name
		meta.Major = major
	}
	if meta.Name != "" && meta.Major != "" {
		return meta
	}

	profileBody, err := s.fetchProfileHTML(ctx)
	if err != nil {
		s.helper.logger.Debug("fetch jw profile page failed", zap.Error(err))
		return meta
	}

	name, major, ok := parseProfileMeta(profileBody)
	if !ok {
		return meta
	}
	if meta.Name == "" {
		meta.Name = name
	}
	if meta.Major == "" {
		meta.Major = major
	}
	return meta
}

func (s *jwSession) fetchProfileHTML(ctx context.Context) (string, error) {
	candidates := []jwPageRequest{
		{
			Name:    "profile_xsgrxx",
			Method:  http.MethodGet,
			Path:    "/xsgrxx/xsxx",
			Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx",
		},
		{
			Name:    "profile_grxx",
			Method:  http.MethodGet,
			Path:    "/grxx/xsxx",
			Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx",
		},
	}
	return s.fetchFirstPage(ctx, candidates, func(body string) bool {
		_, _, ok := parseProfileMeta(body)
		return ok
	})
}

func (s *jwSession) fetchCourseWeekHTML(ctx context.Context, targetDate string) (string, error) {
	return s.fetchFirstPage(ctx, []jwPageRequest{
		{
			Name:    "course_week_post",
			Method:  http.MethodPost,
			Path:    "/framework/main_index_loadkb.jsp",
			Query:   url.Values{"rq": {targetDate}},
			Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx",
		},
		{
			Name:    "course_week_get",
			Method:  http.MethodGet,
			Path:    "/framework/main_index_loadkb.jsp",
			Query:   url.Values{"rq": {targetDate}},
			Referer: s.helper.jsxsdBaseURL + "/framework/xsMain.htmlx",
		},
	}, nil)
}

func (s *jwSession) fetchExamHTML(ctx context.Context, term string) (string, error) {
	return s.fetchFirstPage(ctx, examRequests(s.helper.jsxsdBaseURL, term), nil)
}

func (s *jwSession) fetchExamScoreHTML(ctx context.Context, term string) (string, error) {
	form := url.Values{"kcxz": {""}, "kcmc": {""}, "xsfs": {"all"}}
	if strings.TrimSpace(term) != "" {
		form.Set("kksj", term)
	}

	return s.fetchFirstPage(ctx, []jwPageRequest{
		{
			Name:    "exam_score_post",
			Method:  http.MethodPost,
			Path:    "/kscj/cjcx_list",
			Form:    form,
			Referer: s.helper.jsxsdBaseURL + "/kscj/cjcx_query",
		},
		{
			Name:    "exam_score_get",
			Method:  http.MethodGet,
			Path:    "/kscj/cjcx_list",
			Query:   cloneValues(form),
			Referer: s.helper.jsxsdBaseURL + "/kscj/cjcx_query",
		},
	}, nil)
}

func (s *jwSession) fetchFirstPage(ctx context.Context, candidates []jwPageRequest, accept func(string) bool) (string, error) {
	if accept == nil {
		accept = hasUsefulHTML
	}

	var lastErr error
	for _, candidate := range candidates {
		body, err := s.requestPage(ctx, candidate)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", pageRequestName(candidate), err)
			continue
		}
		if looksLikeJWLoginPage(body) {
			lastErr = fmt.Errorf("%s: jw session expired while requesting %s", pageRequestName(candidate), candidate.Path)
			continue
		}
		if accept(body) {
			return body, nil
		}
		lastErr = fmt.Errorf("%s: page content did not match expected shape", pageRequestName(candidate))
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("jw page fetch failed")
}

func (s *jwSession) requestPage(ctx context.Context, req jwPageRequest) (string, error) {
	endpoint := strings.TrimRight(s.helper.jsxsdBaseURL, "/") + req.Path
	if len(req.Query) > 0 {
		endpoint += "?" + cloneValues(req.Query).Encode()
	}

	switch req.Method {
	case http.MethodPost:
		return s.postForm(ctx, endpoint, req.Form, false, req.Referer)
	default:
		return s.get(ctx, endpoint, req.Referer)
	}
}

func (s *jwSession) get(ctx context.Context, endpoint string, referer string) (string, error) {
	return s.doWithRetry(ctx, http.MethodGet, endpoint, false, nil, referer)
}

func (s *jwSession) postForm(ctx context.Context, endpoint string, form url.Values, ajax bool, referer string) (string, error) {
	return s.doWithRetry(ctx, http.MethodPost, endpoint, ajax, cloneValues(form), referer)
}

func (s *jwSession) doWithRetry(
	ctx context.Context,
	method string,
	endpoint string,
	ajax bool,
	form url.Values,
	referer string,
) (string, error) {
	attempts := s.helper.requestAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := s.newRequest(ctx, method, endpoint, ajax, form, referer)
		if err != nil {
			return "", err
		}

		body, err := s.do(req)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if attempt >= attempts || !shouldRetryJWRequest(err) {
			break
		}

		s.helper.logger.Warn("jw request retry",
			zap.String("method", method),
			zap.String("url", endpoint),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)
		if err := sleepWithContext(ctx, time.Duration(attempt)*defaultJWRetryBackoff); err != nil {
			return "", err
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("jw request failed")
}

func (s *jwSession) newRequest(
	ctx context.Context,
	method string,
	endpoint string,
	ajax bool,
	form url.Values,
	referer string,
) (*http.Request, error) {
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(cloneValues(form).Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	s.applyHeaders(req, ajax, referer)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}
	return req, nil
}

func (s *jwSession) applyHeaders(req *http.Request, ajax bool, referer string) {
	req.Header.Set("User-Agent", s.helper.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
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

	body, err := decodeHTTPResponseBody(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", &jwHTTPStatusError{
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}
	return body, nil
}

func decodeHTTPResponseBody(resp *http.Response) (string, error) {
	if resp == nil || resp.Body == nil {
		return "", nil
	}

	reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		reader = resp.Body
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (h *JWRequestHelper) encryptByDES(password string) (string, error) {
	out, err := encrypt.DESECBEncrypt([]byte(password), []byte(jwLoginPasswordDESKey))
	if err != nil {
		h.logger.Warn("encrypt jw password failed", zap.Error(err))
		return "", fmt.Errorf("encrypt jw password failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(out), nil
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
		{
			Name:    "exam_list_xsks_get",
			Method:  http.MethodGet,
			Path:    "/xsks/xsksap_list",
			Referer: strings.TrimRight(baseURL, "/") + "/framework/xsMain.htmlx",
		},
		{
			Name:    "exam_list_xsksap_get",
			Method:  http.MethodGet,
			Path:    "/xsksap/xsksap_list",
			Referer: strings.TrimRight(baseURL, "/") + "/framework/xsMain.htmlx",
		},
	}
	if strings.TrimSpace(term) == "" {
		return values
	}

	queryNames := []string{"xnxq01id", "xnxqid", "kksj"}
	var out []jwPageRequest
	for _, req := range values {
		out = append(out, req)
		for _, name := range queryNames {
			req.Query = url.Values{name: {term}}
			req.Name = req.Name + "_" + name
			out = append(out, req)

			req.Method = http.MethodPost
			req.Query = nil
			req.Form = url.Values{name: {term}}
			req.Name = req.Name + "_post"
			out = append(out, req)

			req = resetExamRequest(req)
		}
	}
	return out
}

func resetExamRequest(req jwPageRequest) jwPageRequest {
	req.Method = http.MethodGet
	req.Query = nil
	req.Form = nil
	switch req.Path {
	case "/xsks/xsksap_list":
		req.Name = "exam_list_xsks_get"
	case "/xsksap/xsksap_list":
		req.Name = "exam_list_xsksap_get"
	}
	return req
}

func cloneValues(src url.Values) url.Values {
	if len(src) == 0 {
		return nil
	}

	dst := make(url.Values, len(src))
	for key, values := range src {
		if len(values) == 0 {
			dst[key] = []string{}
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}

func pageRequestName(req jwPageRequest) string {
	if strings.TrimSpace(req.Name) != "" {
		return req.Name
	}
	return strings.TrimSpace(req.Method + " " + req.Path)
}

func shouldRetryJWRequest(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var statusErr *jwHTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusTooManyRequests || statusErr.StatusCode >= http.StatusInternalServerError
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Timeout()
	}

	return false
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
