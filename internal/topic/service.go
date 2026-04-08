package topic

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

type Service struct {
	repo     *Repository
	redis    *redis.Client
	logger   *zap.Logger
	producer EventProducer
}

func NewService(
	db *gorm.DB,
	mongoDB *mongo.Database,
	rds *redis.Client,
	_ *config.Config,
	logger *zap.Logger,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		repo:   NewRepository(db, mongoDB),
		redis:  rds,
		logger: logger,
	}
}

func (s *Service) SetProducer(producer EventProducer) {
	s.producer = producer
}

func (s *Service) Create(ctx context.Context, claims *jwtutil.Claims, req *CreateTopicReq) (*Topic, error) {
	if claims == nil {
		return nil, ErrInvalidAuthClaims
	}
	if req == nil {
		return nil, bizerr.Param(errMsgInvalidParam)
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
		Imgs:          ensureSlice(req.Imgs),
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

	if _, err := s.repo.CreateTopic(ctx, topic); err != nil {
		return nil, bizerr.InternalWrap("创建帖子失败", err)
	}
	s.prepareTopic(topic)

	if s.producer != nil {
		if err := s.producer.SendTopicCheck(ctx, TopicCheckMsg{TopicID: topic.ID.Hex()}); err != nil {
			s.logger.Warn("send topic check mq failed", zap.Error(err), zap.String("topicID", topic.ID.Hex()))
		}
	}
	return topic, nil
}

func (s *Service) Delete(ctx context.Context, topicID string, userID int64, isAdmin bool) error {
	if !isAdmin && userID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return err
	}

	ok, err := s.repo.HideTopic(ctx, oid, userIDString(userID), isAdmin)
	if err != nil {
		return bizerr.InternalWrap("删除帖子失败", err)
	}
	if !ok {
		return ErrTopicNotFound
	}

	if s.producer != nil {
		if err := s.producer.SendDeleteTopic(ctx, TopicDeleteMsg{TopicID: topicID}); err != nil {
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
		return nil, ErrTopicNotFound
	}

	if err := s.repo.IncrementTopicField(ctx, topic.ID, "visitedNum", 1); err != nil {
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
	if req == nil || userID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	update := bson.M{}
	if req.Title != "" {
		update["title"] = req.Title
	}
	if req.Content != "" {
		update["content"] = req.Content
	}
	if len(req.Imgs) > 0 {
		update["imgs"] = ensureSlice(req.Imgs)
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

	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return err
	}

	ok, err := s.repo.UpdateTopic(ctx, oid, userIDString(userID), update)
	if err != nil {
		return bizerr.InternalWrap("更新帖子失败", err)
	}
	if !ok {
		return ErrTopicNotFound
	}

	return nil
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, size int) (*PageResult[Topic], error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	return s.listByFilter(
		ctx,
		bson.M{"userId": userIDString(userID), "hasCheck": true},
		page,
		size,
		userIDString(userID),
		bson.D{{Key: "_id", Value: -1}, {Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "visitedNum", Value: -1}},
	)
}

func (s *Service) ListTargetUserTopics(
	ctx context.Context,
	currentUserID, targetUserID int64,
	page, size int,
) (*PageResult[Topic], error) {
	if targetUserID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	author, err := s.repo.FindUserByID(ctx, targetUserID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询目标用户失败", err)
	}
	if author == nil {
		return nil, ErrTargetUserNotFound
	}

	return s.listByFilter(
		ctx,
		bson.M{
			"userId":   userIDString(targetUserID),
			"hasCheck": true,
		},
		page,
		size,
		userIDString(currentUserID),
		bson.D{{Key: "_id", Value: -1}, {Key: "visitedNum", Value: -1}},
	)
}

func (s *Service) listByFilter(
	ctx context.Context,
	filter bson.M,
	page, size int,
	queryUserID string,
	sort bson.D,
) (*PageResult[Topic], error) {
	topics, total, err := s.repo.FindTopicsPage(ctx, filter, sort, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询帖子列表失败", err)
	}

	s.prepareTopics(topics)
	if err := s.fillLikeAndCollection(ctx, queryUserID, topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err))
	}
	return NewPageResult(topics, total, page, size), nil
}
