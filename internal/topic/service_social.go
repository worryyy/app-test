package topic

import (
	"context"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) LikeTopic(ctx context.Context, userID int64, topicID string) error {
	topic, err := s.GetByID(ctx, topicID, "")
	if err != nil {
		return err
	}
	if topic == nil {
		return ErrTopicNotFound
	}

	filter := bson.M{"userId": strconv.FormatInt(userID, 10), "themeName": topic.ThemeID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"userId":      strconv.FormatInt(userID, 10),
			"themeName":   topic.ThemeID,
			"accountType": 1,
		},
		"$addToSet": bson.M{"topicIds": topicID},
	}
	res, err := s.mongoDB.Collection("campus_topic_like").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("like topic: %w", err)
	}
	if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
		_, incErr := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"likeNum": 1}})
		if incErr != nil {
			s.logger.Warn("increase topic like num failed", zap.Error(incErr), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) UnlikeTopic(ctx context.Context, userID int64, topicID string) error {
	filter := bson.M{"userId": strconv.FormatInt(userID, 10)}
	update := bson.M{"$pull": bson.M{"topicIds": topicID}}
	res, err := s.mongoDB.Collection("campus_topic_like").UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("unlike topic: %w", err)
	}
	if res.ModifiedCount > 0 {
		_, incErr := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"likeNum": -1}})
		if incErr != nil {
			s.logger.Warn("decrease topic like num failed", zap.Error(incErr), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListLikedTopics(ctx context.Context, userID int64, page, size int) (*result.CusPage[Topic], error) {
	return s.listTopicsFromArrayDocs(ctx, "campus_topic_like", userID, page, size)
}

func (s *Service) CollectTopic(ctx context.Context, userID int64, topicID string) error {
	topic, err := s.GetByID(ctx, topicID, "")
	if err != nil {
		return err
	}
	if topic == nil {
		return ErrTopicNotFound
	}

	filter := bson.M{"userId": strconv.FormatInt(userID, 10), "themeName": topic.ThemeID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"userId":      strconv.FormatInt(userID, 10),
			"themeName":   topic.ThemeID,
			"accountType": 1,
		},
		"$addToSet": bson.M{"topicIds": topicID},
	}
	res, err := s.mongoDB.Collection("campus_topic_collection").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("collect topic: %w", err)
	}
	if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
		_, incErr := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"collectionNum": 1}})
		if incErr != nil {
			s.logger.Warn("increase topic collection num failed", zap.Error(incErr), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) UncollectTopic(ctx context.Context, userID int64, topicID string) error {
	filter := bson.M{"userId": strconv.FormatInt(userID, 10)}
	update := bson.M{"$pull": bson.M{"topicIds": topicID}}
	res, err := s.mongoDB.Collection("campus_topic_collection").UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("uncollect topic: %w", err)
	}
	if res.ModifiedCount > 0 {
		_, incErr := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"collectionNum": -1}})
		if incErr != nil {
			s.logger.Warn("decrease topic collection num failed", zap.Error(incErr), zap.String("topicID", topicID))
		}
	}
	return nil
}

func (s *Service) ListCollectedTopics(ctx context.Context, userID int64, page, size int) (*result.CusPage[Topic], error) {
	return s.listTopicsFromArrayDocs(ctx, "campus_topic_collection", userID, page, size)
}

func (s *Service) fillLikeAndCollection(ctx context.Context, userID string, topics []Topic) error {
	if userID == "" || len(topics) == 0 {
		return nil
	}

	ids := make(map[string]int, len(topics))
	for i := range topics {
		ids[topics[i].ID.Hex()] = i
		topics[i].HasLike = false
		topics[i].HasCollection = false
	}

	likeCur, err := s.mongoDB.Collection("campus_topic_like").Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return fmt.Errorf("load topic likes: %w", err)
	}
	defer likeCur.Close(ctx)

	var likeDocs []TopicLike
	if err := likeCur.All(ctx, &likeDocs); err != nil {
		return fmt.Errorf("decode topic likes: %w", err)
	}
	for _, d := range likeDocs {
		for _, id := range d.TopicIDs {
			if idx, ok := ids[id]; ok {
				topics[idx].HasLike = true
			}
		}
	}

	colCur, err := s.mongoDB.Collection("campus_topic_collection").Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return fmt.Errorf("load topic collections: %w", err)
	}
	defer colCur.Close(ctx)

	var colDocs []TopicCollection
	if err := colCur.All(ctx, &colDocs); err != nil {
		return fmt.Errorf("decode topic collections: %w", err)
	}
	for _, d := range colDocs {
		for _, id := range d.TopicIDs {
			if idx, ok := ids[id]; ok {
				topics[idx].HasCollection = true
			}
		}
	}
	return nil
}

func (s *Service) listTopicsFromArrayDocs(ctx context.Context, collName string, userID int64, page, size int) (*result.CusPage[Topic], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	type arrayDoc struct {
		TopicIDs []string `bson:"topicIds"`
	}

	cur, err := s.mongoDB.Collection(collName).Find(ctx, bson.M{"userId": strconv.FormatInt(userID, 10)})
	if err != nil {
		return nil, fmt.Errorf("find %s docs: %w", collName, err)
	}
	defer cur.Close(ctx)

	var docs []arrayDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode %s docs: %w", collName, err)
	}

	allIDs := make([]string, 0)
	for _, d := range docs {
		allIDs = append(allIDs, d.TopicIDs...)
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
	return result.NewCusPage(topics, total, page, size), nil
}

func (s *Service) findByIDs(ctx context.Context, topicIDs []string) ([]Topic, error) {
	if len(topicIDs) == 0 {
		return []Topic{}, nil
	}
	oids := make([]primitive.ObjectID, 0, len(topicIDs))
	for _, id := range topicIDs {
		oid, err := primitive.ObjectIDFromHex(id)
		if err == nil {
			oids = append(oids, oid)
		}
	}
	if len(oids) == 0 {
		return []Topic{}, nil
	}

	cur, err := s.topicColl().Find(ctx, bson.M{"_id": bson.M{"$in": oids}, "hasCheck": true})
	if err != nil {
		return nil, fmt.Errorf("find topics by ids: %w", err)
	}
	defer cur.Close(ctx)

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode topics by ids: %w", err)
	}
	for i := range topics {
		topics[i].Imgs = result.EnsureSlice(topics[i].Imgs)
	}
	return topics, nil
}

func mustObjectID(hexID string) primitive.ObjectID {
	oid, _ := primitive.ObjectIDFromHex(hexID)
	return oid
}
