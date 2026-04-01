package topic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

var ErrTopicAlreadyLiked = errors.New("topic already liked")

func (s *Service) LikeTopic(ctx context.Context, claims *jwtutil.Claims, topicID string) error {
	if claims == nil {
		return result.ErrParam
	}

	topic, err := s.getTopicByID(ctx, topicID, true)
	if err != nil {
		return err
	}
	if topic == nil {
		return result.NewBizError(result.CodeNotExisted, "帖子不存在")
	}

	currentUserID := userIDString(claims.UserID)
	themeName := s.resolveThemeName(ctx, topic.ThemeID)
	filter := bson.M{"userId": currentUserID, "themeName": themeName}
	update := bson.M{
		"$setOnInsert": bson.M{
			"userId":      currentUserID,
			"themeName":    themeName,
			"accountType": claims.AccountType,
		},
		"$addToSet": bson.M{"topicIds": topicID},
	}
	res, err := s.mongoDB.Collection("campus_topic_like").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("like topic: %w", err)
	}
	if res.ModifiedCount == 0 && res.UpsertedCount == 0 {
		return ErrTopicAlreadyLiked
	}
	if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
		if _, err := s.topicColl().UpdateByID(ctx, topic.ID, bson.M{"$inc": bson.M{"likeNum": 1}}); err != nil {
			s.logger.Warn("increase topic like num failed", zap.Error(err), zap.String("topicID", topicID))
		}
		s.sendTopicNotify(ctx, topic.UserID, currentUserID, topicID, "点赞了你的帖子", "TOPIC_LIKE")
	}
	return nil
}

func (s *Service) UnlikeTopic(ctx context.Context, userID int64, topicID string) error {
	filter := bson.M{"userId": userIDString(userID)}
	update := bson.M{"$pull": bson.M{"topicIds": topicID}}
	res, err := s.mongoDB.Collection("campus_topic_like").UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("unlike topic: %w", err)
	}
	if res.ModifiedCount > 0 {
		if _, err := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"likeNum": -1}}); err != nil {
			s.logger.Warn("decrease topic like num failed", zap.Error(err), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListLikedTopics(ctx context.Context, userID int64, page, size int) (*result.CusPage[Topic], error) {
	return s.listTopicsFromArrayDocs(ctx, "campus_topic_like", userID, page, size)
}

func (s *Service) CollectTopic(ctx context.Context, claims *jwtutil.Claims, topicID string) error {
	if claims == nil {
		return result.ErrParam
	}

	topic, err := s.getTopicByID(ctx, topicID, true)
	if err != nil {
		return err
	}
	if topic == nil {
		return result.NewBizError(result.CodeNotExisted, "帖子不存在")
	}

	currentUserID := userIDString(claims.UserID)
	themeName := s.resolveThemeName(ctx, topic.ThemeID)

	// Enforce 500-item collection limit (matches Java TopicCollectionImpl)
	collCount, err := s.countCollectionItems(ctx, currentUserID)
	if err != nil {
		return fmt.Errorf("count collections: %w", err)
	}
	if collCount >= 500 {
		return result.NewBizError(result.CodeFail, "收藏数量已达上限")
	}

	filter := bson.M{"userId": currentUserID, "themeName": themeName}
	update := bson.M{
		"$setOnInsert": bson.M{
			"userId":      currentUserID,
			"themeName":    themeName,
			"accountType": claims.AccountType,
		},
		"$addToSet": bson.M{"topicIds": topicID},
	}
	res, err := s.mongoDB.Collection("campus_topic_collection").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("collect topic: %w", err)
	}
	if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
		if _, err := s.topicColl().UpdateByID(ctx, topic.ID, bson.M{"$inc": bson.M{"collectionNum": 1}}); err != nil {
			s.logger.Warn("increase topic collection num failed", zap.Error(err), zap.String("topicID", topicID))
		}
		s.sendTopicNotify(ctx, topic.UserID, currentUserID, topicID, "收藏了你的帖子", "TOPIC_COLLECTION")
	}
	return nil
}

func (s *Service) UncollectTopic(ctx context.Context, userID int64, topicID string) error {
	filter := bson.M{"userId": userIDString(userID)}
	update := bson.M{"$pull": bson.M{"topicIds": topicID}}
	res, err := s.mongoDB.Collection("campus_topic_collection").UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("uncollect topic: %w", err)
	}
	if res.ModifiedCount > 0 {
		if _, err := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"collectionNum": -1}}); err != nil {
			s.logger.Warn("decrease topic collection num failed", zap.Error(err), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListCollectedTopics(ctx context.Context, userID int64, page, size int) (*result.CusPage[Topic], error) {
	return s.listTopicsFromArrayDocs(ctx, "campus_topic_collection", userID, page, size)
}

func (s *Service) listTopicsFromArrayDocs(ctx context.Context, collName string, userID int64, page, size int) (*result.CusPage[Topic], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	cur, err := s.mongoDB.Collection(collName).Find(ctx, bson.M{"userId": userIDString(userID)})
	if err != nil {
		return nil, fmt.Errorf("find %s docs: %w", collName, err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close topic array cursor failed", zap.Error(closeErr), zap.String("collection", collName))
		}
	}()

	var docs []topicStateDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode %s docs: %w", collName, err)
	}

	allIDs := make([]string, 0)
	for _, doc := range docs {
		allIDs = append(allIDs, doc.TopicIDs...)
	}
	total := int64(len(allIDs))
	if len(allIDs) == 0 {
		return result.NewCusPage([]Topic{}, 0, page, size), nil
	}

	start := (page - 1) * size
	if start >= len(allIDs) {
		return result.NewCusPage([]Topic{}, total, page, size), nil
	}
	end := start + size
	if end > len(allIDs) {
		end = len(allIDs)
	}

	topics, err := s.findByIDs(ctx, allIDs[start:end])
	if err != nil {
		return nil, err
	}
	if err := s.fillLikeAndCollection(ctx, userIDString(userID), topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err), zap.String("collection", collName))
	}
	return result.NewCusPage(topics, total, page, size), nil
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

	cur, err := s.topicColl().Find(ctx, bson.M{"_id": bson.M{"$in": oids}, "hasCheck": true})
	if err != nil {
		return nil, fmt.Errorf("find topics by ids: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close topic by ids cursor failed", zap.Error(closeErr))
		}
	}()

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode topics by ids: %w", err)
	}
	s.prepareTopics(topics)

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

	err := s.producer.SendNotifyUser(ctx, mq.NotifyMsg{
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

func (s *Service) countCollectionItems(ctx context.Context, userID string) (int64, error) {
	cur, err := s.mongoDB.Collection("campus_topic_collection").Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return 0, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var total int64
	var docs []topicStateDoc
	if err := cur.All(ctx, &docs); err != nil {
		return 0, err
	}
	for _, doc := range docs {
		total += int64(len(doc.TopicIDs))
	}
	return total, nil
}

func mustObjectID(id string) primitive.ObjectID {
	oid, _ := primitive.ObjectIDFromHex(id)
	return oid
}
