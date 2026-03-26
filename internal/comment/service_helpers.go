package comment

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

const maxPageSize = 100

type userRecord struct {
	ID          int64  `gorm:"column:id"`
	Nickname    string `gorm:"column:nickname"`
	Avatar      string `gorm:"column:avatar"`
	AccountType string `gorm:"column:accountType"`
	Power       int    `gorm:"column:power"`
	Gender      string `gorm:"column:gender"`
	RootUserID  int64  `gorm:"column:rootUserId"`
	Signature   string `gorm:"column:signature"`
}

func (userRecord) TableName() string {
	return "campus_user"
}

func (s *Service) normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = s.defaultPageSize()
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

func (s *Service) loadUser(ctx context.Context, userID int64) (*userRecord, error) {
	if userID <= 0 {
		return nil, result.NewBizError(result.CodeFail, fmt.Sprintf("userId=%d用户不存在", userID))
	}

	var user userRecord
	if err := s.db.WithContext(ctx).Table(user.TableName()).Where("id = ?", userID).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, result.NewBizError(result.CodeFail, fmt.Sprintf("userId=%d用户不存在", userID))
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return &user, nil
}

func (s *Service) loadUserByStringID(ctx context.Context, userID string) (*userRecord, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	if err != nil {
		return nil, nil
	}
	return s.loadUser(ctx, id)
}

func (s *Service) getTopic(ctx context.Context, topicID string, onlyChecked bool) (*CommentTopic, error) {
	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return nil, nil
	}

	filter := bson.M{"_id": oid}
	if onlyChecked {
		filter["hasCheck"] = true
	}

	var topic CommentTopic
	if err := s.mongoDB.Collection("campus_topic").FindOne(ctx, filter).Decode(&topic); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic: %w", err)
	}

	if topic.Imgs == nil {
		topic.Imgs = []string{}
	}
	createdAt := topic.ID.Timestamp()
	topic.CreatedTime = &createdAt
	return &topic, nil
}

func (s *Service) ensureTopicExists(ctx context.Context, topicID string) error {
	topic, err := s.getTopic(ctx, topicID, false)
	if err != nil {
		return err
	}
	if topic == nil {
		return result.NewBizError(result.CodeFail, fmt.Sprintf("%s 帖子不存在", topicID))
	}
	return nil
}

func (s *Service) getCommentByID(ctx context.Context, commentID string) (*Comment, error) {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, result.NewBizError(result.CodeFail, fmt.Sprintf("%s 评论不存在", commentID))
	}

	var cmt Comment
	if err := s.commentColl().FindOne(ctx, bson.M{"_id": oid}).Decode(&cmt); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, result.NewBizError(result.CodeFail, fmt.Sprintf("%s 评论不存在", commentID))
		}
		return nil, fmt.Errorf("find comment: %w", err)
	}
	return &cmt, nil
}

func (s *Service) getHasLikeBatch(ctx context.Context, userID string, comments []Comment) (map[string]struct{}, error) {
	liked := make(map[string]struct{}, len(comments))
	if userID == "" || len(comments) == 0 {
		return liked, nil
	}

	ids := make([]string, 0, len(comments))
	for _, comment := range comments {
		if !comment.ID.IsZero() {
			ids = append(ids, comment.ID.Hex())
		}
	}
	if len(ids) == 0 {
		return liked, nil
	}

	cur, err := s.mongoDB.Collection("campus_comment_like").Find(
		ctx,
		bson.M{"commentId": bson.M{"$in": ids}, "userIds": userID},
		options.Find().SetProjection(bson.M{"commentId": 1}),
	)
	if err != nil {
		return nil, fmt.Errorf("find liked comments: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close comment like cursor failed", zap.Error(closeErr))
		}
	}()

	var rows []struct {
		CommentID string `bson:"commentId"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode liked comments: %w", err)
	}
	for _, row := range rows {
		if row.CommentID != "" {
			liked[row.CommentID] = struct{}{}
		}
	}
	return liked, nil
}

func buildCommentUser(user *userRecord) CommentUser {
	if user == nil {
		return CommentUser{}
	}
	return CommentUser{
		UserID:      strconv.FormatInt(user.ID, 10),
		Avatar:      user.Avatar,
		NickName:    user.Nickname,
		Gender:      user.Gender,
		AccountType: user.AccountType,
		Signature:   user.Signature,
	}
}

func normalizeCommentAccountType(accountType string) string {
	switch strings.ToLower(strings.TrimSpace(accountType)) {
	case "official":
		return "official"
	case "anonymous":
		return "anonymous"
	default:
		return "base"
	}
}

func (s *Service) validateCommentPermission(ctx context.Context, actor *userRecord, topic *CommentTopic) error {
	if actor == nil || topic == nil {
		return nil
	}

	actorType := normalizeCommentAccountType(actor.AccountType)
	topicType := normalizeCommentAccountType(topic.AccountType)
	if actorType == "anonymous" && topicType != "anonymous" {
		return result.NewBizError(result.CodeForbidden, "匿名用户禁止评论非匿名帖")
	}
	if actorType != "base" || topicType != "anonymous" {
		return nil
	}

	topicAuthor, err := s.loadUserByStringID(ctx, topic.UserID)
	if err != nil || topicAuthor == nil {
		return err
	}
	actorRootUserID := actor.RootUserID
	if actorRootUserID == 0 {
		actorRootUserID = actor.ID
	}
	topicRootUserID := topicAuthor.RootUserID
	if topicRootUserID == 0 {
		topicRootUserID = topicAuthor.ID
	}
	if actorRootUserID == topicRootUserID {
		return result.NewBizError(result.CodeForbidden, "禁止左右脑互搏和自导自演")
	}
	return nil
}
