package school

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type JWClient struct {
	helper *JWRequestHelper
}

type JWCommonResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
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
	return &JWClient{
		helper: NewJWRequestHelper(cfg, logger),
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
	query.Set("weeks", strconv.Itoa(week))
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
	payload any,
) (*JWCommonResp, error) {
	if j == nil || j.helper == nil {
		return nil, ErrJWHelperUnavailable
	}
	return j.helper.DoJSON(ctx, method, path, query, payload)
}
