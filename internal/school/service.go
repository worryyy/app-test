package school

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/school/jw_client"

)

var (
	ErrCurrentTermNotConfigured = errors.New("current term not configured")
	ErrCurrentTermInvalid       = errors.New("current term invalid")
)

type Service struct {
	db       *gorm.DB
	mongoDB  *mongo.Database
	redis    *redis.Client
	cfg      *config.Config
	logger   *zap.Logger
	producer *mq.Producer
	jwClient *JWClient
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger, producer *mq.Producer) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:       db,
		mongoDB:  mongoDB,
		redis:    rds,
		cfg:      cfg,
		logger:   logger,
		producer: producer,
		jwClient: NewJWClient(cfg, logger),
	}
}


func (s *Service) GetCourseByWeeks(ctx context.Context, req UserCourseReq) (*JWCommonResp, error) {
	if s.jwClient == nil {
		return nil, errors.New("jw client not initialized")
	}
	return s.jwClient.GetCourseByWeeks(ctx, req.StartDate, req.Week, JWGetCourseReq{
		Term:     req.Term,
		SchoolID: req.SchoolID,
		Password: req.Password,
	})
}

func (s *Service) GetExam(ctx context.Context, req ExamReq) (*JWCommonResp, error) {
	if s.jwClient == nil {
		return nil, errors.New("jw client not initialized")
	}
	return s.jwClient.GetExam(ctx, JWGetExamReq{
		SchoolID: req.SchoolID,
		Password: req.Password,
		XNXQID:   req.XNXQID,
	})
}

func (s *Service) GetExamScore(ctx context.Context, req ExamScoreReq) (*JWCommonResp, error) {
	if s.jwClient == nil {
		return nil, errors.New("jw client not initialized")
	}
	return s.jwClient.GetExamScore(ctx, JWGetExamScoreReq{
		SchoolID: req.SchoolID,
		Password: req.Password,
		SS:       req.SS,
	})
}



func (s *Service) Authenticate(
	ctx context.Context,
	userID int64,
	req AuthenticationReq,
) (*JWLoginData, error) {
	loginResp, err := s.checkJWLogin(ctx, req.SchoolID, req.Password)
	if err != nil {
		return nil, err
	}

	encPwd, err := s.encryptAES(req.Password)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SaveAuthentication(ctx, userID, req, loginResp, encPwd); err != nil {
		return nil, err
	}
	return loginResp, nil
}

func (s *Service) ReAuthentication(
	ctx context.Context,
	userID int64,
	req AuthenticationReq,
) (*JWLoginData, error) {
	return s.Authenticate(ctx, userID, req)
}
