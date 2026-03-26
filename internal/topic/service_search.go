package topic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	cronjob "github.com/Milchstrassse/Ecampus-go/internal/cron"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) Search(
	ctx context.Context,
	userID string,
	themeIDs []string,
	content string,
	page, size, ordCreated int,
) (*result.CusPage[Topic], error) {
	if ordCreated == 0 {
		return s.searchHot(ctx, userID, themeIDs, content, page, size)
	}

	filter := bson.M{"hasCheck": true}
	if len(themeIDs) > 0 {
		filter["themeId"] = bson.M{"$in": themeIDs}
	}
	if strings.TrimSpace(content) != "" {
		filter["content"] = primitive.Regex{Pattern: ".*" + content + ".*", Options: "i"}
	}
	return s.listByFilter(ctx, filter, page, size, userID, bson.D{{Key: "_id", Value: -1}, {Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "visitedNum", Value: -1}})
}

func (s *Service) searchHot(
	ctx context.Context,
	userID string,
	themeIDs []string,
	content string,
	page, size int,
) (*result.CusPage[Topic], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	match := bson.M{"hasCheck": true}
	if len(themeIDs) > 0 {
		match["themeId"] = bson.M{"$in": themeIDs}
	}
	if strings.TrimSpace(content) != "" {
		match["content"] = primitive.Regex{Pattern: ".*" + content + ".*", Options: "i"}
	}

	sevenDaysAgo := primitive.NewObjectIDFromTimestamp(time.Now().AddDate(0, 0, -7))
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$addFields", Value: bson.M{
			"hotScore": bson.M{"$add": []interface{}{
				bson.M{"$multiply": []interface{}{"$commentNum", 9}},
				bson.M{"$multiply": []interface{}{"$likeNum", 6}},
				bson.M{"$multiply": []interface{}{"$visitedNum", 1}},
			}},
		}}},
		{{Key: "$addFields", Value: bson.M{
			"recentFlag": bson.M{"$cond": bson.M{
				"if":   bson.M{"$gte": []interface{}{"$_id", sevenDaysAgo}},
				"then": 1,
				"else": 0,
			}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "recentFlag", Value: -1}, {Key: "hotScore", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$skip", Value: int64((page - 1) * size)}},
		{{Key: "$limit", Value: int64(size)}},
	}

	total, err := s.topicColl().CountDocuments(ctx, match)
	if err != nil {
		return nil, fmt.Errorf("count hot topics: %w", err)
	}

	cur, err := s.topicColl().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate hot topics: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close hot topic cursor failed", zap.Error(closeErr))
		}
	}()

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode hot topics: %w", err)
	}
	s.prepareTopics(topics)
	if err := s.fillLikeAndCollection(ctx, userID, topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err))
	}
	return result.NewCusPage(topics, total, page, size), nil
}

func (s *Service) GetSuggestList(ctx context.Context, userID string, page, size int) (*SuggestList, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	cacheKey := fmt.Sprintf("%d_%d", page, size)
	if cached, err := s.redis.HGet(ctx, rediskey.SuggestTopicListKey, cacheKey).Result(); err == nil && cached != "" {
		var vo SuggestList
		if unmarshalErr := json.Unmarshal([]byte(cached), &vo); unmarshalErr == nil {
			return &vo, nil
		}
	}

	curKey, _ := s.redis.Get(ctx, rediskey.SuggestCurKey).Result()
	if curKey == "" {
		curKey, _ = s.redis.Get(ctx, rediskey.SuggestPrevKey).Result()
	}
	if curKey == "" {
		return &SuggestList{Total: 0, CurPage: page, Size: size, Data: []Topic{}}, nil
	}

	total, err := s.redis.ZCard(ctx, curKey).Result()
	if err != nil {
		return nil, fmt.Errorf("zcard suggest key: %w", err)
	}

	start := int64((page - 1) * size)
	topicIDs, err := s.redis.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   curKey,
		Start: start,
		Stop:  start + int64(size) - 1,
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange suggest key: %w", err)
	}

	topics, err := s.findByIDs(ctx, topicIDs)
	if err != nil {
		return nil, err
	}
	if err := s.fillLikeAndCollection(ctx, userID, topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err))
	}

	vo := &SuggestList{
		Total:   total,
		CurPage: page,
		Size:    size,
		Data:    topics,
	}
	if data, marshalErr := json.Marshal(vo); marshalErr == nil {
		if hsetErr := s.redis.HSet(ctx, rediskey.SuggestTopicListKey, cacheKey, string(data)).Err(); hsetErr != nil {
			s.logger.Warn("cache suggest list failed", zap.Error(hsetErr))
		}
	}
	return vo, nil
}

func (s *Service) RefreshSuggest(ctx context.Context) (int64, error) {
	job := cronjob.NewSuggestJob(s.mongoDB, s.redis, s.logger)
	count, err := job.Generate(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}
