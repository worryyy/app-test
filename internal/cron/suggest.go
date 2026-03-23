package cron

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

type SuggestJob struct {
	mongoDB *mongo.Database
	rds     *redis.Client
	logger  *zap.Logger
}

type suggestTheme struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	NeedSuggest       bool               `bson:"needSuggest"`
	SuggestBasicScore int64              `bson:"suggestBasicScore"`
	SuggestNumber     int                `bson:"suggestNumber"`
	SuggestSetName    string             `bson:"suggestSetName"`
}

type suggestTopic struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	CommentNum int64              `bson:"commentNum"`
	LikeNum    int64              `bson:"likeNum"`
	VisitedNum int64              `bson:"visitedNum"`
}

func NewSuggestJob(mongoDB *mongo.Database, rds *redis.Client, logger *zap.Logger) *SuggestJob {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SuggestJob{mongoDB: mongoDB, rds: rds, logger: logger}
}

func (j *SuggestJob) Generate(ctx context.Context) error {
	if j.mongoDB == nil || j.rds == nil {
		return nil
	}

	themes, err := j.loadSuggestThemes(ctx)
	if err != nil {
		return err
	}
	if len(themes) == 0 {
		return nil
	}

	now := time.Now()
	themeKeys := make([]string, 0, len(themes))
	for _, theme := range themes {
		rankKey := rediskey.SuggestRank(theme.SuggestSetName)
		if err := j.rds.Del(ctx, rankKey).Err(); err != nil {
			return fmt.Errorf("clear suggest key %s: %w", rankKey, err)
		}
		if err := j.buildThemeRank(ctx, theme, rankKey, now); err != nil {
			return err
		}
		themeKeys = append(themeKeys, rankKey)
	}

	version, err := j.rds.Incr(ctx, rediskey.SuggestCountKey).Result()
	if err != nil {
		return fmt.Errorf("increase suggest count: %w", err)
	}
	newKey := fmt.Sprintf("rank:all_%d_%s", version, now.Format("2006-01-02"))

	if err := j.rds.ZUnionStore(ctx, newKey, &redis.ZStore{Keys: themeKeys}).Err(); err != nil {
		return fmt.Errorf("merge suggest rank: %w", err)
	}
	prevKey, _ := j.rds.Get(ctx, rediskey.SuggestCurKey).Result()
	_ = j.rds.Set(ctx, rediskey.SuggestPrevKey, prevKey, 0).Err()
	_ = j.rds.Set(ctx, rediskey.SuggestCurKey, newKey, 0).Err()
	_ = j.rds.Del(ctx, rediskey.SuggestTopicListKey).Err()
	return nil
}

func (j *SuggestJob) CleanupOldAllRanks(ctx context.Context) error {
	if j.rds == nil {
		return nil
	}
	curKey, _ := j.rds.Get(ctx, rediskey.SuggestCurKey).Result()
	prevKey, _ := j.rds.Get(ctx, rediskey.SuggestPrevKey).Result()

	iter := j.rds.Scan(ctx, 0, "rank:all_*", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if key == curKey || key == prevKey {
			continue
		}
		if err := j.rds.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("delete old suggest key %s: %w", key, err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan old suggest keys: %w", err)
	}
	return nil
}

func (j *SuggestJob) loadSuggestThemes(ctx context.Context) ([]suggestTheme, error) {
	cur, err := j.mongoDB.Collection("campus_theme").Find(ctx, bson.M{"needSuggest": true})
	if err != nil {
		return nil, fmt.Errorf("find suggest themes: %w", err)
	}
	defer cur.Close(ctx)

	var themes []suggestTheme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode suggest themes: %w", err)
	}
	return themes, nil
}

func (j *SuggestJob) buildThemeRank(
	ctx context.Context,
	theme suggestTheme,
	rankKey string,
	now time.Time,
) error {
	if strings.TrimSpace(theme.SuggestSetName) == "" {
		return nil
	}
	cur, err := j.mongoDB.Collection("campus_topic").Find(ctx, bson.M{
		"themeId":  theme.ID.Hex(),
		"hasCheck": true,
	}, options.Find().SetProjection(bson.M{
		"_id":        1,
		"commentNum": 1,
		"likeNum":    1,
		"visitedNum": 1,
	}))
	if err != nil {
		return fmt.Errorf("find suggest topics: %w", err)
	}
	defer cur.Close(ctx)

	var topics []suggestTopic
	if err := cur.All(ctx, &topics); err != nil {
		return fmt.Errorf("decode suggest topics: %w", err)
	}
	zs := make([]redis.Z, 0, len(topics))
	for _, topic := range topics {
		score := float64(theme.SuggestBasicScore)
		days := now.Sub(topic.ID.Timestamp()).Hours() / 24
		score -= days * 5
		score += float64(topic.CommentNum) * 10
		score += float64(topic.LikeNum) * 10
		score += float64(topic.VisitedNum) * 3
		zs = append(zs, redis.Z{Score: score, Member: topic.ID.Hex()})
	}
	if len(zs) > 0 {
		if err := j.rds.ZAdd(ctx, rankKey, zs...).Err(); err != nil {
			return fmt.Errorf("zadd suggest rank %s: %w", rankKey, err)
		}
	}
	limit := theme.SuggestNumber
	if limit <= 0 {
		limit = 500
	}
	if err := j.rds.ZRemRangeByRank(ctx, rankKey, 0, int64(-(limit + 1))).Err(); err != nil {
		return fmt.Errorf("trim suggest rank %s: %w", rankKey, err)
	}
	return nil
}
