package school

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

type Service struct {
	repo     *Repository
	cfg      *config.Config
	logger   *zap.Logger
	jwClient *JWClient
}

func NewService(
	db *gorm.DB,
	mongoDB *mongo.Database,
	_ *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		repo:     NewRepository(db, mongoDB),
		cfg:      cfg,
		logger:   logger,
		jwClient: NewJWClient(cfg, logger),
	}
}

func (s *Service) CurTerm(ctx context.Context) (*CurDateAndTerm, error) {
	current, err := s.currentTermRecord(ctx)
	if err != nil {
		return nil, err
	}

	termValue := ""
	if current != nil {
		termValue = strings.TrimSpace(current.Term)
	}
	if termValue == "" {
		return nil, bizerr.NotFound("current term is not configured")
	}

	term, err := s.repo.FindTermByValue(ctx, termValue)
	if err != nil {
		return nil, bizerr.InternalWrap("query term failed", err)
	}
	if term == nil {
		return nil, bizerr.NotFound("current term record is invalid")
	}

	return &CurDateAndTerm{
		CurDate:    time.Now().Format("2006-01-02"),
		CurTerm:    termValue,
		TotalWeeks: term.TotalWeeks,
		StartDate:  term.StartDate,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, userID int64, req AuthenticationReq) (*JWCommonResp, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	loginResp, err := s.checkJWLogin(ctx, req.SchoolID, req.Password)
	if err != nil {
		return nil, err
	}

	_, name, major, err := decodeJWLoginMeta(loginResp.Data)
	if err != nil {
		return nil, bizerr.InternalWrap("decode authentication result failed", err)
	}

	encPwd, err := s.encryptAES(req.Password)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveAuthentication(ctx, userID, req, encPwd, name, major); err != nil {
		return nil, bizerr.InternalWrap("save authentication failed", err)
	}
	return loginResp, nil
}

func (s *Service) ReAuthentication(ctx context.Context, userID int64, req AuthenticationReq) (*JWCommonResp, error) {
	return s.Authenticate(ctx, userID, req)
}

func (s *Service) GetCourseByWeeks(ctx context.Context, req UserCourseReq) (*JWCommonResp, error) {
	resp, err := s.jwClient.GetCourseByWeeks(ctx, req.StartDate, req.Week, JWGetCourseReq{
		Term:     req.Term,
		SchoolID: req.SchoolID,
		Password: req.Password,
	})
	if err != nil {
		return nil, bizerr.InternalWrap("query course failed", err)
	}
	if err := ensureJWRespSuccess(resp, "query course failed"); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) GetExam(ctx context.Context, req ExamReq) (*JWCommonResp, error) {
	resp, err := s.jwClient.GetExam(ctx, JWGetExamReq{
		SchoolID: req.SchoolID,
		Password: req.Password,
		XNXQID:   req.XNXQID,
	})
	if err != nil {
		return nil, bizerr.InternalWrap("query exam failed", err)
	}
	if err := ensureJWRespSuccess(resp, "query exam failed"); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) GetExamScore(ctx context.Context, req ExamScoreReq) (*JWCommonResp, error) {
	resp, err := s.jwClient.GetExamScore(ctx, JWGetExamScoreReq{
		SchoolID: req.SchoolID,
		Password: req.Password,
		SS:       req.SS,
	})
	if err != nil {
		return nil, bizerr.InternalWrap("query exam score failed", err)
	}
	if err := ensureJWRespSuccess(resp, "query exam score failed"); err != nil {
		return nil, err
	}
	return resp, nil
}
