package agentchat

import (
	"context"
	"time"

	agentv1 "github.com/Milchstrassse/Ecampus-go/internal/agentchat/agentv1"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const (
	defaultConversationPageSize    = 15
	maxConversationPageSize        = 100
	defaultTimeout                 = 30 * time.Second
	defaultRateLimitPerMinute      = 10
	defaultMaxConcurrentPerUser    = 2
	defaultMaxConcurrentPerSession = 1
	defaultHistoryPageSize         = 20
	maxHistoryPageSize             = 100
)

type agentClient interface {
	HandleTurn(context.Context, *agentv1.HandleTurnRequest) (*agentv1.HandleTurnResponse, error)
	StreamHandleTurn(context.Context, *agentv1.HandleTurnRequest) (agentv1.AgentService_StreamHandleTurnClient, error)
	GetSessionHistory(context.Context, *agentv1.GetSessionHistoryRequest) (*agentv1.GetSessionHistoryResponse, error)
	DeleteSession(context.Context, *agentv1.DeleteSessionRequest) (*agentv1.DeleteSessionResponse, error)
}

type Service struct {
	db     *gorm.DB
	client agentClient
	redis  *redis.Client
	cfg    *config.Config
	logger *zap.Logger
}

type TurnInput struct {
	RequestID      string
	ConversationID string
	Content        string
	RootUserID     int64
	CurrentUserID  int64
	AccountType    string
	ClientPlatform string
	SchoolID       string
	SchoolName     string
	Locale         string
}

func NewService(db *gorm.DB, rds *redis.Client, cfg *config.Config, logger *zap.Logger, client agentClient) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:     db,
		client: client,
		redis:  rds,
		cfg:    cfg,
		logger: logger,
	}
}

func (s *Service) enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Agent.Enabled && s.client != nil
}

func (s *Service) timeout() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.Agent.TimeoutMS <= 0 {
		return defaultTimeout
	}
	return time.Duration(s.cfg.Agent.TimeoutMS) * time.Millisecond
}

func (s *Service) rateLimitPerMinute() int {
	if s == nil || s.cfg == nil || s.cfg.Agent.RateLimitPerMinute <= 0 {
		return defaultRateLimitPerMinute
	}
	return s.cfg.Agent.RateLimitPerMinute
}

func (s *Service) maxConcurrentPerUser() int {
	if s == nil || s.cfg == nil || s.cfg.Agent.MaxConcurrentPerUser <= 0 {
		return defaultMaxConcurrentPerUser
	}
	return s.cfg.Agent.MaxConcurrentPerUser
}

func (s *Service) maxConcurrentPerConversation() int {
	if s == nil || s.cfg == nil || s.cfg.Agent.MaxConcurrentPerConversation <= 0 {
		return defaultMaxConcurrentPerSession
	}
	return s.cfg.Agent.MaxConcurrentPerConversation
}

func (s *Service) guardTTL() time.Duration {
	ttl := s.timeout() * 2
	if ttl < 2*time.Minute {
		return 2 * time.Minute
	}
	return ttl
}
