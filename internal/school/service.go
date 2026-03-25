package school

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
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

func (s *Service) TermList(ctx context.Context) ([]Term, error) {
	cur, err := s.mongoDB.Collection("campus_term").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("find terms: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close term cursor failed", zap.Error(closeErr))
		}
	}()

	var list []Term
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode terms: %w", err)
	}
	return list, nil
}

func (s *Service) CurrentTerm(ctx context.Context) (*CurDateAndTermVO, error) {
	var cur CurTerm
	err := s.mongoDB.Collection("campus_cur_term").FindOne(ctx, bson.M{}).Decode(&cur)
	if err == mongo.ErrNoDocuments {
		return nil, ErrCurrentTermNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("find current term: %w", err)
	}

	var term Term
	if err := s.mongoDB.Collection("campus_term").FindOne(ctx, bson.M{"term": cur.Term}).Decode(&term); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrCurrentTermInvalid
		}
		return nil, fmt.Errorf("find term detail: %w", err)
	}

	return &CurDateAndTermVO{
		CurDate:    time.Now().Format("2006-01-02"),
		CurTerm:    cur.Term,
		StartDate:  term.StartDate,
		TotalWeeks: term.TotalWeeks,
	}, nil
}

func (s *Service) AddTerm(ctx context.Context, term *Term) (string, error) {
	res, err := s.mongoDB.Collection("campus_term").InsertOne(ctx, term)
	if err != nil {
		return "", fmt.Errorf("add term: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("term id invalid")
	}
	return oid.Hex(), nil
}

func (s *Service) DeleteTerm(ctx context.Context, termID string) error {
	oid, err := primitive.ObjectIDFromHex(termID)
	if err != nil {
		return fmt.Errorf("invalid term id: %w", err)
	}
	_, err = s.mongoDB.Collection("campus_term").DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("delete term: %w", err)
	}
	return nil
}

func (s *Service) SetCurrentTerm(ctx context.Context, termID string) error {
	oid, err := primitive.ObjectIDFromHex(termID)
	if err != nil {
		return fmt.Errorf("invalid term id: %w", err)
	}

	var term Term
	if err := s.mongoDB.Collection("campus_term").FindOne(ctx, bson.M{"_id": oid}).Decode(&term); err != nil {
		if err == mongo.ErrNoDocuments {
			return result.ErrNotExisted
		}
		return fmt.Errorf("find term before set current: %w", err)
	}

	_, err = s.mongoDB.Collection("campus_cur_term").UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"term": term.Term}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("set current term: %w", err)
	}
	return nil
}

func (s *Service) SetCourseColor(ctx context.Context, userID int64, colors map[string]string) error {
	data, err := json.Marshal(colors)
	if err != nil {
		return fmt.Errorf("marshal course colors: %w", err)
	}
	key := fmt.Sprintf("campus:courseColor:%d", userID)
	if err := s.redis.Set(ctx, key, string(data), 0).Err(); err != nil {
		return fmt.Errorf("set course color: %w", err)
	}
	return nil
}

func (s *Service) GetCourseColor(ctx context.Context, userID int64) (map[string]string, error) {
	key := fmt.Sprintf("campus:courseColor:%d", userID)
	raw, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get course color: %w", err)
	}
	var colors map[string]string
	if err := json.Unmarshal([]byte(raw), &colors); err != nil {
		return nil, fmt.Errorf("unmarshal course colors: %w", err)
	}
	return colors, nil
}

func (s *Service) RequestGetCourse(ctx context.Context, userID int64, stuNum, stuPwd, term string, week int) error {
	if s.producer == nil {
		return nil
	}
	msg := mq.CourseMsg{
		UserID: userID,
		StuNum: stuNum,
		StuPwd: stuPwd,
		Term:   term,
		Week:   week,
	}
	if err := s.producer.SendGetCourse(ctx, msg); err != nil {
		return fmt.Errorf("send get course mq: %w", err)
	}
	return nil
}

func (s *Service) GetCourseByWeeks(ctx context.Context, userID int64, term string, weeks []int) ([]UserCourse, error) {
	var list []UserCourse
	if err := s.db.WithContext(ctx).Where("userId = ? AND term = ? AND week IN ?", userID, term, weeks).Order("week ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("query user courses by weeks: %w", err)
	}
	return list, nil
}

func (s *Service) ParseWeek(v string) int {
	week, err := strconv.Atoi(v)
	if err != nil || week <= 0 {
		return 1
	}
	return week
}

func (s *Service) ToCusPage(list []Course, total int64, page, size int) *result.CusPage[Course] {
	return result.NewCusPage(list, total, page, size)
}
