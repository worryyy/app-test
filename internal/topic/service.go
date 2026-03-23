package topic

import (
	"context"
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
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

var (
	ErrTopicNotFound = errors.New("topic not found")
)

type Service struct {
	db       *gorm.DB
	mongoDB  *mongo.Database
	redis    *redis.Client
	cfg      *config.Config
	logger   *zap.Logger
	producer *mq.Producer
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
	}
}

func (s *Service) Create(ctx context.Context, claims *jwtutil.Claims, req *CreateTopicReq) (string, error) {
	if claims == nil || req == nil {
		return "", result.ErrParam
	}

	topic := Topic{
		ThemeID:       req.ThemeID,
		UserID:        strconv.FormatInt(claims.UserID, 10),
		Title:         req.Title,
		Content:       req.Content,
		Imgs:          result.EnsureSlice(req.Imgs),
		HasCheck:      false,
		VisitedNum:    0,
		LikeNum:       0,
		CommentNum:    0,
		CollectionNum: 0,
		Ext:           req.Ext,
		AccountType:   mapAccountType(claims.AccountType),
		NickName:      req.NickName,
		Avatar:        req.Avatar,
	}

	res, err := s.topicColl().InsertOne(ctx, topic)
	if err != nil {
		return "", fmt.Errorf("insert topic: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", errors.New("inserted topic id type invalid")
	}
	topicID := oid.Hex()
	if s.producer != nil {
		sendErr := s.producer.SendTopicCheck(ctx, mq.TopicCheckMsg{TopicID: topicID})
		if sendErr != nil {
			s.logger.Warn("send topic check mq failed", zap.Error(sendErr), zap.String("topicID", topicID))
		}
	}
	return topicID, nil
}

func (s *Service) Delete(ctx context.Context, topicID string, userID int64, isAdmin bool) error {
	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return fmt.Errorf("invalid topic id: %w", err)
	}

	filter := bson.M{"_id": oid}
	if !isAdmin {
		filter["userId"] = strconv.FormatInt(userID, 10)
	}
	res, err := s.topicColl().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"hasCheck": false}})
	if err != nil {
		return fmt.Errorf("delete topic: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrTopicNotFound
	}

	if s.producer != nil {
		sendErr := s.producer.SendDelTopicSearch(ctx, mq.AddTopicSearchMsg{TopicID: topicID})
		if sendErr != nil {
			s.logger.Warn("send delete search mq failed", zap.Error(sendErr), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, topicID string, queryUserID string) (*Topic, error) {
	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return nil, fmt.Errorf("invalid topic id: %w", err)
	}

	var topic Topic
	err = s.topicColl().FindOne(ctx, bson.M{"_id": oid, "hasCheck": true}).Decode(&topic)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query topic by id: %w", err)
	}

	_, incErr := s.topicColl().UpdateByID(ctx, oid, bson.M{"$inc": bson.M{"visitedNum": 1}})
	if incErr != nil {
		s.logger.Warn("increase topic visited num failed", zap.Error(incErr), zap.String("topicID", topicID))
	}

	if queryUserID != "" {
		if fillErr := s.fillLikeAndCollection(ctx, queryUserID, []Topic{topic}); fillErr != nil {
			s.logger.Warn("fill like/collection failed", zap.Error(fillErr), zap.String("topicID", topicID))
		}
	}
	topic.Imgs = result.EnsureSlice(topic.Imgs)
	return &topic, nil
}

func (s *Service) Update(ctx context.Context, topicID string, userID int64, req *CreateTopicReq) error {
	if req == nil {
		return result.ErrParam
	}
	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return fmt.Errorf("invalid topic id: %w", err)
	}

	update := bson.M{
		"title":   req.Title,
		"content": req.Content,
		"imgs":    result.EnsureSlice(req.Imgs),
		"ext":     req.Ext,
	}

	res, err := s.topicColl().UpdateOne(ctx, bson.M{
		"_id":    oid,
		"userId": strconv.FormatInt(userID, 10),
	}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("update topic: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrTopicNotFound
	}

	if s.producer != nil {
		sendErr := s.producer.SendUpdateTopicSearch(ctx, mq.AddTopicSearchMsg{TopicID: topicID})
		if sendErr != nil {
			s.logger.Warn("send update search mq failed", zap.Error(sendErr), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, size int) (*result.CusPage[Topic], error) {
	return s.listByFilter(ctx, bson.M{"userId": strconv.FormatInt(userID, 10), "hasCheck": true}, page, size)
}

func (s *Service) ListByTheme(ctx context.Context, userID int64, themeID string, page, size int) (*result.CusPage[Topic], error) {
	return s.listByFilter(ctx, bson.M{
		"userId":   strconv.FormatInt(userID, 10),
		"themeId":  themeID,
		"hasCheck": true,
	}, page, size)
}

func (s *Service) ListTargetUserTopics(ctx context.Context, targetUserID int64, page, size int) (*result.CusPage[Topic], error) {
	return s.listByFilter(ctx, bson.M{"userId": strconv.FormatInt(targetUserID, 10), "hasCheck": true}, page, size)
}

func (s *Service) ListFollowTopics(ctx context.Context, currentUserID int64, page, size int) (*result.CusPage[Topic], error) {
	followCur, err := s.mongoDB.Collection("campus_follow").Find(ctx, bson.M{"followerId": strconv.FormatInt(currentUserID, 10)})
	if err != nil {
		return nil, fmt.Errorf("find followings: %w", err)
	}
	defer followCur.Close(ctx)

	type followDoc struct {
		FollowingID string `bson:"followingId"`
	}
	var followings []followDoc
	if err := followCur.All(ctx, &followings); err != nil {
		return nil, fmt.Errorf("decode followings: %w", err)
	}

	ids := make([]string, 0, len(followings))
	for _, f := range followings {
		if f.FollowingID != "" {
			ids = append(ids, f.FollowingID)
		}
	}
	if len(ids) == 0 {
		return result.NewCusPage([]Topic{}, 0, page, size), nil
	}
	return s.listByFilter(ctx, bson.M{"userId": bson.M{"$in": ids}, "hasCheck": true}, page, size)
}

func (s *Service) listByFilter(ctx context.Context, filter bson.M, page, size int) (*result.CusPage[Topic], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	total, err := s.topicColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count topics: %w", err)
	}

	opts := options.Find().
		SetSort(bson.M{"_id": -1}).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cur, err := s.topicColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find topics: %w", err)
	}
	defer cur.Close(ctx)

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode topics: %w", err)
	}

	for i := range topics {
		topics[i].Imgs = result.EnsureSlice(topics[i].Imgs)
	}
	return result.NewCusPage(topics, total, page, size), nil
}

func (s *Service) topicColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_topic")
}

func mapAccountType(accountType string) int {
	switch accountType {
	case "official":
		return 2
	case "anonymous":
		return 3
	default:
		return 1
	}
}

func nowISODate() string {
	return time.Now().Format("2006-01-02")
}
