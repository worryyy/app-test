package topic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) Search(ctx context.Context, userID, themeID, keyword string, page, size int, orderBy string) (*result.CusPage[Topic], error) {
	if orderBy == "hot" {
		return s.SearchHot(ctx, userID, themeID, page, size)
	}

	if keyword != "" {
		ids, total, err := s.SearchByKeyword(ctx, keyword, "", page, size)
		if err != nil {
			return nil, err
		}
		topics, err := s.findByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		if err := s.fillLikeAndCollection(ctx, userID, topics); err != nil {
			s.logger.Warn("fill topic like/collection failed", zap.Error(err))
		}
		return result.NewCusPage(topics, total, page, size), nil
	}

	filter := bson.M{"hasCheck": true}
	if themeID != "" {
		filter["themeId"] = themeID
	}
	return s.listByFilter(ctx, filter, page, size)
}

func (s *Service) SearchHot(ctx context.Context, userID, themeID string, page, size int) (*result.CusPage[Topic], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	match := bson.M{"hasCheck": true}
	if themeID != "" {
		match["themeId"] = themeID
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
			"isRecent": bson.M{"$gte": []interface{}{"$_id", sevenDaysAgo}},
		}}},
		{{Key: "$sort", Value: bson.M{"isRecent": -1, "hotScore": -1, "_id": -1}}},
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
	defer cur.Close(ctx)

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode hot topics: %w", err)
	}
	if err := s.fillLikeAndCollection(ctx, userID, topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err))
	}
	for i := range topics {
		topics[i].Imgs = result.EnsureSlice(topics[i].Imgs)
	}
	return result.NewCusPage(topics, total, page, size), nil
}

func (s *Service) SearchByKeyword(ctx context.Context, keyword, themeName string, page, size int) ([]string, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	searchStr := tokenizeForSearch(keyword)
	filter := bson.M{"$text": bson.M{"$search": searchStr}}
	if themeName != "" {
		filter["themeName"] = themeName
	}

	coll := s.mongoDB.Collection("campus_topic_search")
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count search docs: %w", err)
	}

	opts := options.Find().
		SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}, "topicId": 1}).
		SetSort(bson.M{"score": bson.M{"$meta": "textScore"}}).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("find search docs: %w", err)
	}
	defer cur.Close(ctx)

	var rows []TopicSearch
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, fmt.Errorf("decode search docs: %w", err)
	}

	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.TopicID)
	}
	return ids, total, nil
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
	topicIDs, err := s.redis.ZRevRange(ctx, curKey, start, start+int64(size)-1).Result()
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

func tokenizeForSearch(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return keyword
	}
	tokens := strings.Fields(keyword)
	if len(tokens) == 0 {
		return keyword
	}
	return strings.Join(tokens, " ")
}

func (s *Service) RefreshSuggest(ctx context.Context) (int64, error) {
	if s.redis == nil {
		return time.Now().Unix(), nil
	}
	v, err := s.redis.Incr(ctx, rediskey.SuggestCountKey).Result()
	if err != nil {
		return 0, fmt.Errorf("refresh suggest version: %w", err)
	}
	return v, nil
}
