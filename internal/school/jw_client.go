package school

import (
	"context"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
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
	if j == nil || j.helper == nil {
		return nil, ErrJWHelperUnavailable
	}
	return j.helper.CheckLogin(ctx, schoolID, password)
}

func (j *JWClient) GetCourseByWeeks(
	ctx context.Context,
	startDate string,
	week int,
	req JWGetCourseReq,
) (*JWCommonResp, error) {
	if j == nil || j.helper == nil {
		return nil, ErrJWHelperUnavailable
	}
	return j.helper.GetCourseByWeeks(ctx, startDate, week, req)
}

func (j *JWClient) GetExam(ctx context.Context, req JWGetExamReq) (*JWCommonResp, error) {
	if j == nil || j.helper == nil {
		return nil, ErrJWHelperUnavailable
	}
	return j.helper.GetExam(ctx, req)
}

func (j *JWClient) GetExamScore(ctx context.Context, req JWGetExamScoreReq) (*JWCommonResp, error) {
	if j == nil || j.helper == nil {
		return nil, ErrJWHelperUnavailable
	}
	return j.helper.GetExamScore(ctx, req)
}
