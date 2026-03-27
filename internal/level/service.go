package level

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	db      *gorm.DB
	mongoDB *mongo.Database
	redis   *redis.Client
	cfg     *config.Config
	logger  *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:      db,
		mongoDB: mongoDB,
		redis:   rds,
		cfg:     cfg,
		logger:  logger,
	}
}

func (s *Service) SignIn(ctx context.Context, userID int64) error {
	yearMonth := time.Now().Format("200601")
	day := int64(time.Now().Day() - 1)
	key := userSignKey(userID, yearMonth)

	already, err := s.redis.GetBit(ctx, key, day).Result()
	if err != nil {
		return fmt.Errorf("check sign status: %w", err)
	}
	if already == 1 {
		return result.NewBizError(result.CodeFail, "今日已签到")
	}

	if err := s.redis.SetBit(ctx, key, day, 1).Err(); err != nil {
		return fmt.Errorf("set sign bit: %w", err)
	}
	return nil
}

func (s *Service) GetSignDetail(ctx context.Context, userID int64) (*UserSignDetail, error) {
	yearMonth := time.Now().Format("200601")
	key := userSignKey(userID, yearMonth)
	day := time.Now().Day()

	todaySigned, err := s.redis.GetBit(ctx, key, int64(day-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("get today sign status: %w", err)
	}

	exp, err := s.GetExp(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserSignDetail{
		UserID:  userID,
		UserExp: exp,
		Signed:  todaySigned == 1,
	}, nil
}

func (s *Service) GetExp(ctx context.Context, userID int64) (int, error) {
	return s.getExpWithCache(ctx, userID)
}

func (s *Service) GetExpBatch(ctx context.Context, userIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(userIDs))
	for _, userID := range userIDs {
		exp, err := s.getExpWithCache(ctx, userID)
		if err != nil {
			return nil, err
		}
		out[userID] = exp
	}
	return out, nil
}

func (s *Service) getExpWithCache(ctx context.Context, userID int64) (int, error) {
	key := fmt.Sprintf("campus:userExp:%d", userID)
	cached, err := s.redis.Get(ctx, key).Result()
	if err == nil && cached != "" {
		v, convErr := strconv.Atoi(cached)
		if convErr == nil {
			return v, nil
		}
	}

	var sum int64
	if err := s.db.WithContext(ctx).Model(&ExpDetail{}).
		Select("COALESCE(SUM(get_exp), 0)").
		Where("user_id = ?", userID).
		Scan(&sum).Error; err != nil {
		return 0, fmt.Errorf("query user exp sum: %w", err)
	}

	ttl := 30 * time.Minute
	if sum == 0 {
		ttl = 5 * time.Second
	}
	if setErr := s.redis.Set(ctx, key, sum, ttl).Err(); setErr != nil {
		s.logger.Warn("cache user exp failed", zap.Error(setErr), zap.Int64("userID", userID))
	}
	return int(sum), nil
}

func (s *Service) TestAOP(context.Context) string {
	return "这是/testAop接口"
}

func (s *Service) ExpPlus3(context.Context, int64) {}

func userSignKey(userID int64, yearMonth string) string {
	return fmt.Sprintf("campus:userSign:%d:%s", userID, yearMonth)
}
