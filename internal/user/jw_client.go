package user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type JWClient struct {
	cfg    *config.Config
	logger *zap.Logger
	client *http.Client
}

type JWCommonResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type JWLoginData struct {
	IsLogin bool   `json:"is_login"`
	Major   string `json:"major"`
	Name    string `json:"name"`
}

type JWLoginReq struct {
	SchoolID string `json:"school_id"`
	Password string `json:"password"`
}

type JWGetCourseReq struct {
	Term     string `json:"term"`
	SchoolID string `json:"school_id"`
	Password string `json:"password"`
}

type JWGetExamReq struct {
	SchoolID string `json:"school_id"`
	Password string `json:"password"`
	XNXQID   string `json:"xnxqid"`
}

type JWGetExamScoreReq struct {
	SchoolID string `json:"school_id"`
	Password string `json:"password"`
	SS       string `json:"ss"`
}

func NewJWClient(cfg *config.Config, logger *zap.Logger) *JWClient {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &JWClient{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (j *JWClient) CheckLogin(ctx context.Context, schoolID, password string) (*JWCommonResp, error) {
	return j.doJSON(ctx, http.MethodPost, "/check_login", nil, JWLoginReq{
		SchoolID: schoolID,
		Password: password,
	})
}

func (j *JWClient) GetCourseByWeeks(
	ctx context.Context,
	startDate string,
	week int,
	req JWGetCourseReq,
) (*JWCommonResp, error) {
	query := url.Values{}
	query.Set("date", startDate)
	query.Set("weeks", fmt.Sprintf("%d", week))
	return j.doJSON(ctx, http.MethodPost, "/get_course_by_weeks", query, req)
}

func (j *JWClient) GetExam(ctx context.Context, req JWGetExamReq) (*JWCommonResp, error) {
	return j.doJSON(ctx, http.MethodPost, "/get_exam", nil, req)
}

func (j *JWClient) GetExamScore(ctx context.Context, req JWGetExamScoreReq) (*JWCommonResp, error) {
	return j.doJSON(ctx, http.MethodPost, "/get_exam_score", nil, req)
}

func (j *JWClient) doJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	payload interface{},
) (*JWCommonResp, error) {
	if j.cfg == nil || strings.TrimSpace(j.cfg.JW.BaseURL) == "" {
		return nil, fmt.Errorf("jw base url not configured")
	}

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal jw request: %w", err)
		}
	}

	endpoint := strings.TrimRight(j.cfg.JW.BaseURL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create jw request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(j.cfg.JW.APIKey) != "" && j.cfg.JW.APIKey != "replace-me" {
		req.Header.Set("X-API-Key", j.cfg.JW.APIKey)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request jw api: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			j.logger.Warn("close jw api response body failed", zap.Error(closeErr))
		}
	}()

	var commonResp JWCommonResp
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("decode jw response: %w", err)
	}
	return &commonResp, nil
}
