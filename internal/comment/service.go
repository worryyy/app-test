package comment

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
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
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

func (s *Service) AddComment(ctx context.Context, topicID string, user CommentUser, content, parentCmtID, rootCmtID string, isAuthor bool) (string, error) {
	cmt := Comment{
		TopicID:     topicID,
		Comment:     content,
		CreatedTime: time.Now(),
		User:        user,
		ParentCmtID: parentCmtID,
		RootCmtID:   rootCmtID,
		IsAuthor:    isAuthor,
		LikeNum:     0,
		CommentNum:  0,
		HasCheck:    false,
	}

	res, err := s.commentColl().InsertOne(ctx, cmt)
	if err != nil {
		return "", fmt.Errorf("insert comment: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", errors.New("comment inserted id type invalid")
	}

	cmt.ID = oid
	if s.producer != nil {
		sendErr := s.producer.SendAddComment(ctx, mq.AddCommentMsg{Comment: cmt})
		if sendErr != nil {
			s.logger.Warn("send add comment mq failed", zap.Error(sendErr), zap.String("commentID", oid.Hex()))
		}
	}
	return oid.Hex(), nil
}

func (s *Service) DeleteComment(ctx context.Context, topicID, commentID string, userID int64, isAdmin bool) error {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return fmt.Errorf("invalid comment id: %w", err)
	}

	filter := bson.M{"_id": oid, "topicId": topicID}
	if !isAdmin {
		filter["user.userId"] = strconv.FormatInt(userID, 10)
	}

	res, err := s.commentColl().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"hasCheck": false}})
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (s *Service) ListByTopic(ctx context.Context, topicID string, page, size int) (*result.CusPage[Comment], error) {
	return s.listByFilter(ctx, bson.M{"topicId": topicID, "hasCheck": true}, page, size)
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, size int) (*result.CusPage[Comment], error) {
	return s.listByFilter(ctx, bson.M{"user.userId": strconv.FormatInt(userID, 10), "hasCheck": true}, page, size)
}

func (s *Service) ListTargetUserComments(ctx context.Context, targetUserID int64, page, size int) (*result.CusPage[Comment], error) {
	return s.listByFilter(ctx, bson.M{"user.userId": strconv.FormatInt(targetUserID, 10), "hasCheck": true}, page, size)
}

func (s *Service) LikeComment(ctx context.Context, commentID string, userID int64) error {
	filter := bson.M{"commentId": commentID}
	update := bson.M{
		"$setOnInsert": bson.M{"commentId": commentID},
		"$addToSet":    bson.M{"userIds": strconv.FormatInt(userID, 10)},
	}
	res, err := s.mongoDB.Collection("campus_comment_like").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("like comment: %w", err)
	}
	if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
		if err := s.incCommentLike(ctx, commentID, 1); err != nil {
			s.logger.Warn("increase comment like num failed", zap.Error(err), zap.String("commentID", commentID))
		}
	}
	return nil
}

func (s *Service) UnlikeComment(ctx context.Context, commentID string, userID int64) error {
	res, err := s.mongoDB.Collection("campus_comment_like").UpdateOne(
		ctx,
		bson.M{"commentId": commentID},
		bson.M{"$pull": bson.M{"userIds": strconv.FormatInt(userID, 10)}},
	)
	if err != nil {
		return fmt.Errorf("unlike comment: %w", err)
	}
	if res.ModifiedCount > 0 {
		if err := s.incCommentLike(ctx, commentID, -1); err != nil {
			s.logger.Warn("decrease comment like num failed", zap.Error(err), zap.String("commentID", commentID))
		}
	}
	return nil
}

func (s *Service) listByFilter(ctx context.Context, filter bson.M, page, size int) (*result.CusPage[Comment], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	total, err := s.commentColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count comments: %w", err)
	}

	opts := options.Find().
		SetSort(bson.M{"createdTime": -1}).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cur, err := s.commentColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find comments: %w", err)
	}
	defer cur.Close(ctx)

	var comments []Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, fmt.Errorf("decode comments: %w", err)
	}
	return result.NewCusPage(comments, total, page, size), nil
}

func (s *Service) incCommentLike(ctx context.Context, commentID string, delta int64) error {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return fmt.Errorf("invalid comment id: %w", err)
	}
	_, err = s.commentColl().UpdateByID(ctx, oid, bson.M{"$inc": bson.M{"likeNum": delta}})
	if err != nil {
		return fmt.Errorf("update comment like num: %w", err)
	}
	return nil
}

func (s *Service) commentColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_comment")
}
