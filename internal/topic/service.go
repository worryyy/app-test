package topic

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"

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

func (s *Service) Create(ctx context.Context, claims *jwtutil.Claims, req *CreateTopicReq) (*Topic, error) {
	if claims == nil || req == nil {
		return nil, result.ErrParam
	}
	if err := s.ensureThemeExists(ctx, req.ThemeID); err != nil {
		return nil, err
	}
	author, err := s.resolveTopicAuthor(ctx, claims, req.AccountType)
	if err != nil {
		return nil, err
	}

	title := req.Title
	if title == "" {
		title = " "
	}

	topic := &Topic{
		ThemeID:       req.ThemeID,
		UserID:        userIDString(author.ID),
		Title:         title,
		Content:       req.Content,
		Imgs:          result.EnsureSlice(req.Imgs),
		HasCheck:      false,
		VisitedNum:    0,
		LikeNum:       0,
		CommentNum:    0,
		CollectionNum: 0,
		Ext:           req.Ext,
		AccountType:   author.AccountType,
		NickName:      author.Nickname,
		Avatar:        author.Avatar,
		HasLike:       false,
		HasCollection: false,
	}

	res, err := s.topicColl().InsertOne(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("insert topic: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, errors.New("inserted topic id type invalid")
	}
	topic.ID = oid
	s.prepareTopic(topic)

	if s.producer != nil {
		sendErr := s.producer.SendTopicCheck(ctx, mq.TopicCheckMsg{TopicID: oid.Hex()})
		if sendErr != nil {
			s.logger.Warn("send topic check mq failed", zap.Error(sendErr), zap.String("topicID", oid.Hex()))
		}
	}
	return topic, nil
}

func (s *Service) Delete(ctx context.Context, topicID string, userID int64, isAdmin bool) error {
	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return fmt.Errorf("invalid topic id: %w", err)
	}

	filter := bson.M{"_id": oid}
	if !isAdmin {
		filter["userId"] = userIDString(userID)
	}
	res, err := s.topicColl().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"hasCheck": false}})
	if err != nil {
		return fmt.Errorf("delete topic: %w", err)
	}
	if res.MatchedCount == 0 {
		return result.NewBizError(result.CodeNotExisted, "帖子不存在")
	}

	if s.producer != nil {
		if err := s.producer.SendDelTopicSearch(ctx, mq.AddTopicSearchMsg{TopicID: topicID}); err != nil {
			s.logger.Warn("send delete search mq failed", zap.Error(err), zap.String("topicID", topicID))
		}
		if err := s.producer.SendDeleteTopic(ctx, mq.TopicDeleteMsg{TopicID: topicID}); err != nil {
			s.logger.Warn("send delete topic mq failed", zap.Error(err), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, topicID string, queryUserID string) (*Topic, error) {
	topic, err := s.getTopicByID(ctx, topicID, false)
	if err != nil {
		return nil, err
	}
	if topic == nil {
		return nil, result.NewBizError(result.CodeNotExisted, "帖子不存在")
	}

	if _, err := s.topicColl().UpdateByID(ctx, topic.ID, bson.M{"$inc": bson.M{"visitedNum": 1}}); err != nil {
		s.logger.Warn("increase topic visited num failed", zap.Error(err), zap.String("topicID", topicID))
	}
	topics := []Topic{*topic}
	if err := s.fillLikeAndCollection(ctx, queryUserID, topics); err != nil {
		s.logger.Warn("fill like/collection failed", zap.Error(err), zap.String("topicID", topicID))
	}
	*topic = topics[0]
	return topic, nil
}

func (s *Service) Update(ctx context.Context, topicID string, userID int64, req *UpdateTopicReq) error {
	if req == nil {
		return result.ErrParam
	}

	update := bson.M{}
	if req.Title != "" {
		update["title"] = req.Title
	}
	if req.Content != "" {
		update["content"] = req.Content
	}
	if len(req.Imgs) > 0 {
		update["imgs"] = result.EnsureSlice(req.Imgs)
	}
	if req.Ext != nil && *req.Ext != "" {
		update["ext"] = *req.Ext
	}
	if len(update) == 0 && req.HasCheck == nil {
		return nil
	}
	if req.HasCheck != nil {
		update["hasCheck"] = *req.HasCheck
	} else {
		update["hasCheck"] = false
	}

	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return fmt.Errorf("invalid topic id: %w", err)
	}
	res, err := s.topicColl().UpdateOne(ctx, bson.M{
		"_id":    oid,
		"userId": userIDString(userID),
	}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("update topic: %w", err)
	}
	if res.MatchedCount == 0 {
		return result.NewBizError(result.CodeNotExisted, "帖子不存在")
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
	return s.listByFilter(ctx, bson.M{"userId": userIDString(userID), "hasCheck": true}, page, size, userIDString(userID), bson.D{{Key: "_id", Value: -1}, {Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "visitedNum", Value: -1}})
}

func (s *Service) ListByTheme(ctx context.Context, userID int64, themeID string, page, size int) (*result.CusPage[Topic], error) {
	if err := s.ensureThemeExists(ctx, themeID); err != nil {
		return nil, err
	}
	return s.listByFilter(ctx, bson.M{
		"userId":   userIDString(userID),
		"themeId":  themeID,
		"hasCheck": true,
	}, page, size, userIDString(userID), bson.D{{Key: "_id", Value: -1}, {Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "visitedNum", Value: -1}})
}

func (s *Service) ListTargetUserTopics(ctx context.Context, currentUserID, targetUserID int64, page, size int) (*result.CusPage[Topic], error) {
	var author topicAuthor
	err := s.db.WithContext(ctx).Where("id = ?", targetUserID).First(&author).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, result.NewBizError(result.CodeNotExisted, "目标用户不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("query target user: %w", err)
	}
	return s.listByFilter(ctx, bson.M{
		"userId":   userIDString(targetUserID),
		"hasCheck": true,
	}, page, size, userIDString(currentUserID), bson.D{{Key: "_id", Value: -1}, {Key: "visitedNum", Value: -1}})
}

func (s *Service) ListFollowTopics(ctx context.Context, currentUserID int64, page, size int) (*result.CusPage[Topic], error) {
	followCur, err := s.mongoDB.Collection("campus_follow").Find(ctx, bson.M{"followerId": currentUserID})
	if err != nil {
		return nil, fmt.Errorf("find followings: %w", err)
	}
	defer func() {
		if closeErr := followCur.Close(ctx); closeErr != nil {
			s.logger.Warn("close follow cursor failed", zap.Error(closeErr))
		}
	}()

	type followDoc struct {
		FollowingID int64 `bson:"followingId"`
	}
	var followings []followDoc
	if err := followCur.All(ctx, &followings); err != nil {
		return nil, fmt.Errorf("decode followings: %w", err)
	}

	ids := make([]string, 0, len(followings))
	for _, follow := range followings {
		if follow.FollowingID != 0 {
			ids = append(ids, strconv.FormatInt(follow.FollowingID, 10))
		}
	}
	if len(ids) == 0 {
		return result.NewCusPage([]Topic{}, 0, page, size), nil
	}

	return s.listByFilter(ctx, bson.M{
		"userId":   bson.M{"$in": ids},
		"hasCheck": true,
	}, page, size, userIDString(currentUserID), bson.D{{Key: "_id", Value: -1}, {Key: "visitedNum", Value: -1}})
}

func (s *Service) listByFilter(
	ctx context.Context,
	filter bson.M,
	page, size int,
	queryUserID string,
	sort bson.D,
) (*result.CusPage[Topic], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if len(sort) == 0 {
		sort = bson.D{{Key: "_id", Value: -1}}
	}

	total, err := s.topicColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count topics: %w", err)
	}

	opts := options.Find().
		SetSort(sort).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cur, err := s.topicColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find topics: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close topic cursor failed", zap.Error(closeErr))
		}
	}()

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode topics: %w", err)
	}
	s.prepareTopics(topics)
	if err := s.fillLikeAndCollection(ctx, queryUserID, topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err))
	}
	return result.NewCusPage(topics, total, page, size), nil
}

func (s *Service) topicColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_topic")
}
