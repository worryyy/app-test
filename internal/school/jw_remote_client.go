package school

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const (
	jwRemoteTimeout        = 50 * time.Second
	jwRemoteMaxConnections = 500
)

type jwRemoteClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	logger  *zap.Logger
}

type jwCommonRespWire struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Msg     string `json:"msg"`
	Data    any    `json:"data"`
}

func newJWRemoteClient(cfg *config.Config, logger *zap.Logger) *jwRemoteClient {
	if cfg == nil || strings.TrimSpace(cfg.JW.BaseURL) == "" {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: jwRemoteTimeout, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          jwRemoteMaxConnections,
		MaxIdleConnsPerHost:   jwRemoteMaxConnections,
		MaxConnsPerHost:       jwRemoteMaxConnections,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   jwRemoteTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &jwRemoteClient{
		baseURL: strings.TrimRight(cfg.JW.BaseURL, "/"),
		apiKey:  strings.TrimSpace(cfg.JW.APIKey),
		client: &http.Client{
			Timeout:   jwRemoteTimeout,
			Transport: transport,
		},
		logger: logger,
	}
}

func (c *jwRemoteClient) CheckLogin(ctx context.Context, schoolID, password string) (*JWCommonResp, error) {
	return c.doJSON(ctx, http.MethodPost, "/check_login", nil, JWLoginReq{
		SchoolID: schoolID,
		Password: password,
	})
}

func (c *jwRemoteClient) GetCourseByWeeks(
	ctx context.Context,
	startDate string,
	week int,
	req JWGetCourseReq,
) (*JWCommonResp, error) {
	query := url.Values{
		"date":  {strings.TrimSpace(startDate)},
		"weeks": {strconv.Itoa(week)},
	}
	return c.doJSON(ctx, http.MethodPost, "/get_course_by_weeks", query, req)
}

func (c *jwRemoteClient) GetExam(ctx context.Context, req JWGetExamReq) (*JWCommonResp, error) {
	return c.doJSON(ctx, http.MethodPost, "/get_exam", nil, req)
}

func (c *jwRemoteClient) GetExamScore(ctx context.Context, req JWGetExamScoreReq) (*JWCommonResp, error) {
	return c.doJSON(ctx, http.MethodPost, "/get_exam_score", nil, req)
}

func (c *jwRemoteClient) doJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
) (*JWCommonResp, error) {
	if c == nil || c.client == nil {
		return nil, ErrJWClientUnavailable
	}

	endpoint, err := c.resolveURL(path, query)
	if err != nil {
		return nil, err
	}

	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal jw request body: %w", err)
		}
		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("create jw request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request jw service: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jw response: %w", err)
	}

	parsed, decodeErr := decodeJWCommonResp(raw)
	if decodeErr == nil {
		normalizeJWCommonResp(parsed, resp.StatusCode)
		if resp.StatusCode >= http.StatusBadRequest {
			c.logger.Warn("jw service returned non-2xx status",
				zap.Int("status", resp.StatusCode),
				zap.String("path", path),
				zap.String("message", parsed.Message),
			)
		}
		return parsed, nil
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("jw service status %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("decode jw response: %w", decodeErr)
}

func (c *jwRemoteClient) resolveURL(path string, query url.Values) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse jw base url: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint := base
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint.String(), nil
}

func decodeJWCommonResp(raw []byte) (*JWCommonResp, error) {
	var wire jwCommonRespWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, err
	}

	return &JWCommonResp{
		Code:    wire.Code,
		Message: firstNonBlank(wire.Message, wire.Msg),
		Data:    wire.Data,
	}, nil
}

func normalizeJWCommonResp(resp *JWCommonResp, statusCode int) {
	if resp == nil {
		return
	}
	if resp.Code == 0 {
		resp.Code = statusCode
	}
	if strings.TrimSpace(resp.Message) == "" {
		resp.Message = http.StatusText(statusCode)
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
