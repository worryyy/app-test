package topic

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

func (s *Service) LikeTopic(ctx context.Context, claims *jwtutil.Claims, topicID string) error {
	if claims == nil {
		return ErrInvalidAuthClaims
	}

	topic, err := s.getTopicByID(ctx, topicID, true)
	if err != nil {
		return err
	}
	if topic == nil {
		return ErrTopicNotFound
	}

	currentUserID := userIDString(claims.UserID)
	themeName := s.resolveThemeName(ctx, topic.ThemeID)
	added, err := s.repo.AddTopicState(ctx, mongoCollTopicLike, currentUserID, themeName, claims.AccountType, topicID)
	if err != nil {
		return bizerr.InternalWrap("点赞帖子失败", err)
	}
	if !added {
		return ErrTopicAlreadyLiked
	}

	if err := s.repo.IncrementTopicField(ctx, topic.ID, "likeNum", 1); err != nil {
		s.logger.Warn("increase topic like num failed", zap.Error(err), zap.String("topicID", topicID))
	}
	s.sendTopicNotify(ctx, topic.UserID, currentUserID, topicID, "点赞了你的帖子", "TOPIC_LIKE")
	return nil
}

func (s *Service) UnlikeTopic(ctx context.Context, userID int64, accountType, topicID string) error {
	if userID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return err
	}

	removed, err := s.repo.RemoveTopicState(ctx, mongoCollTopicLike, userIDString(userID), accountType, topicID)
	if err != nil {
		return bizerr.InternalWrap("取消点赞失败", err)
	}
	if removed {
		if err := s.repo.IncrementTopicField(ctx, oid, "likeNum", -1); err != nil {
			s.logger.Warn("decrease topic like num failed", zap.Error(err), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListLikedTopics(ctx context.Context, userID int64, accountType string, page, size int) (*PageResult[Topic], error) {
	return s.listTopicsFromArrayDocs(ctx, mongoCollTopicLike, userID, accountType, page, size)
}

func (s *Service) CollectTopic(ctx context.Context, claims *jwtutil.Claims, topicID string) error {
	if claims == nil {
		return ErrInvalidAuthClaims
	}

	topic, err := s.getTopicByID(ctx, topicID, true)
	if err != nil {
		return err
	}
	if topic == nil {
		return ErrTopicNotFound
	}

	currentUserID := userIDString(claims.UserID)
	themeName := s.resolveThemeName(ctx, topic.ThemeID)

	collCount, err := s.repo.CountTopicStateItems(ctx, mongoCollTopicCollection, currentUserID, claims.AccountType)
	if err != nil {
		return bizerr.InternalWrap("查询收藏数量失败", err)
	}
	if collCount >= 500 {
		return ErrTopicCollectionLimit
	}

	added, err := s.repo.AddTopicState(ctx, mongoCollTopicCollection, currentUserID, themeName, claims.AccountType, topicID)
	if err != nil {
		return bizerr.InternalWrap("收藏帖子失败", err)
	}
	if added {
		if err := s.repo.IncrementTopicField(ctx, topic.ID, "collectionNum", 1); err != nil {
			s.logger.Warn("increase topic collection num failed", zap.Error(err), zap.String("topicID", topicID))
		}
		s.sendTopicNotify(ctx, topic.UserID, currentUserID, topicID, "收藏了你的帖子", "TOPIC_COLLECTION")
	}
	return nil
}

func (s *Service) UncollectTopic(ctx context.Context, userID int64, accountType, topicID string) error {
	if userID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return err
	}

	removed, err := s.repo.RemoveTopicState(ctx, mongoCollTopicCollection, userIDString(userID), accountType, topicID)
	if err != nil {
		return bizerr.InternalWrap("取消收藏失败", err)
	}
	if removed {
		if err := s.repo.IncrementTopicField(ctx, oid, "collectionNum", -1); err != nil {
			s.logger.Warn("decrease topic collection num failed", zap.Error(err), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListCollectedTopics(ctx context.Context, userID int64, accountType string, page, size int) (*PageResult[Topic], error) {
	return s.listTopicsFromArrayDocs(ctx, mongoCollTopicCollection, userID, accountType, page, size)
}

func (s *Service) listTopicsFromArrayDocs(ctx context.Context, collName string, userID int64, accountType string, page, size int) (*PageResult[Topic], error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	docs, err := s.repo.FindTopicStateDocs(ctx, collName, userIDString(userID), accountType)
	if err != nil {
		return nil, bizerr.InternalWrap("查询帖子列表失败", err)
	}

	if docs == nil || len(docs.TopicIDs) == 0 {
		return NewPageResult([]Topic{}, 0, page, size), nil
	}

	allIDs := make([]string, 0, len(docs.TopicIDs))
	for _, topicID := range docs.TopicIDs {
		allIDs = append(allIDs, topicID)
	}

	total := int64(len(allIDs))
	if len(allIDs) == 0 {
		return NewPageResult([]Topic{}, 0, page, size), nil
	}

	start := (page - 1) * size
	if start >= len(allIDs) {
		return NewPageResult([]Topic{}, total, page, size), nil
	}
	end := start + size
	if end > len(allIDs) {
		end = len(allIDs)
	}

	topics, err := s.findByIDs(ctx, allIDs[start:end])
	if err != nil {
		return nil, bizerr.InternalWrap("查询帖子列表失败", err)
	}
	indexes := resetTopicFlags(topics)
	switch collName {
	case mongoCollTopicLike:
		for i := range topics {
			topics[i].HasLike = true
		}
		if err := s.fillTopicFlags(ctx, userIDString(userID), accountType, indexes, topics, mongoCollTopicCollection, false); err != nil {
			s.logger.Warn("fill topic collection flags failed", zap.Error(err), zap.String("collection", collName))
		}
	case mongoCollTopicCollection:
		for i := range topics {
			topics[i].HasCollection = true
		}
		if err := s.fillTopicFlags(ctx, userIDString(userID), accountType, indexes, topics, mongoCollTopicLike, true); err != nil {
			s.logger.Warn("fill topic like flags failed", zap.Error(err), zap.String("collection", collName))
		}
	default:
		if err := s.fillLikeAndCollection(ctx, userIDString(userID), accountType, topics); err != nil {
			s.logger.Warn("fill topic like/collection failed", zap.Error(err), zap.String("collection", collName))
		}
	}
	return NewPageResult(topics, total, page, size), nil
}

func (s *Service) findByIDs(ctx context.Context, topicIDs []string) ([]Topic, error) {
	if len(topicIDs) == 0 {
		return []Topic{}, nil
	}

	oids := make([]primitive.ObjectID, 0, len(topicIDs))
	for _, id := range topicIDs {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		oids = append(oids, oid)
	}
	if len(oids) == 0 {
		return []Topic{}, nil
	}

	topics, err := s.repo.FindTopicsByIDs(ctx, oids, true)
	if err != nil {
		return nil, err
	}
	s.prepareTopics(topics)
	if err := s.fillTopicUserCertification(ctx, topics); err != nil {
		return nil, err
	}

	ordered := make([]Topic, 0, len(topics))
	byID := make(map[string]Topic, len(topics))
	for _, topic := range topics {
		byID[topic.ID.Hex()] = topic
	}
	for _, id := range topicIDs {
		topic, ok := byID[id]
		if ok {
			ordered = append(ordered, topic)
		}
	}
	return ordered, nil
}

func (s *Service) sendTopicNotify(ctx context.Context, targetUserID, senderUserID, topicID, content, notifyType string) {
	if s.producer == nil || targetUserID == "" || targetUserID == senderUserID {
		return
	}

	err := s.producer.SendNotifyUser(ctx, NotifyMsg{
		TargetUserID: targetUserID,
		SenderUserID: senderUserID,
		Type:         notifyType,
		Content:      content,
		TopicID:      topicID,
		CreatedTime:  time.Now(),
	})
	if err != nil {
		s.logger.Warn("send topic notify failed", zap.Error(err), zap.String("topicID", topicID), zap.String("type", notifyType))
	}
}

func isTopicAlreadyLiked(err error) bool {
	return errors.Is(err, ErrTopicAlreadyLiked)
}
