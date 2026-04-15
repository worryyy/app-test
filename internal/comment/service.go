package comment

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
)

const maxPageSize = 100

type Service struct {
	repo            *Repository
	cfg             *config.Config
	logger          *zap.Logger
	producer        CommentProducer
	sensitiveFilter sensitive.Filter
}

type CommentProducer interface {
	SendAddComment(ctx context.Context, cmt Comment) error
	SendDeleteComment(ctx context.Context, topicID, commentID string) error
}

func NewService(
	db *gorm.DB,
	mongoDB *mongo.Database,
	_ *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
	producer CommentProducer,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:     NewRepository(db, mongoDB),
		cfg:      cfg,
		logger:   logger,
		producer: producer,
	}
}

func (s *Service) SetSensitiveFilter(filter sensitive.Filter) {
	s.sensitiveFilter = filter
}

func (s *Service) AddComment(ctx context.Context, topicID string, currentUserID int64, content, parentCmtID string) (string, error) {
	topic, err := s.getTopic(ctx, topicID, false)
	if err != nil {
		return "", err
	}
	if topic == nil {
		return "", ErrTopicNotFound
	}

	user, err := s.loadUser(ctx, currentUserID)
	if err != nil {
		return "", err
	}
	if err := s.validateCommentPermission(ctx, user, topic); err != nil {
		return "", err
	}

	filteredContent, err := s.filterText(ctx, content)
	if err != nil {
		return "", err
	}

	comment := Comment{
		TopicID:     topicID,
		Comment:     filteredContent,
		CreatedTime: time.Now(),
		User:        buildCommentUser(user),
		ParentCmtID: parentCmtID,
		RootCmtID:   DefaultRootCommentID,
		IsAuthor:    topic.UserID == strconv.FormatInt(currentUserID, 10),
		LikeNum:     0,
		CommentNum:  0,
	}

	if parentCmtID != DefaultRootCommentID {
		parent, err := s.getCommentByID(ctx, parentCmtID)
		if err != nil {
			return "", err
		}
		parentUser := parent.User
		comment.Parent = &parentUser
		if parent.RootCmtID != "" && parent.RootCmtID != DefaultRootCommentID {
			comment.RootCmtID = parent.RootCmtID
		} else {
			comment.RootCmtID = parent.ID.Hex()
		}
	}

	if _, err := s.repo.CreateComment(ctx, &comment); err != nil {
		return "", bizerr.InternalWrap("创建评论失败", err)
	}

	if s.producer != nil {
		if err := s.producer.SendAddComment(ctx, comment); err != nil {
			s.logger.Warn("send add comment mq failed", zap.Error(err), zap.String("commentID", comment.ID.Hex()))
		}
	}
	return comment.ID.Hex(), nil
}

func (s *Service) DeleteComment(ctx context.Context, topicID, commentID string, userID int64) error {
	if userID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	comment, err := s.getCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	commentOID, err := parseObjectID(commentID)
	if err != nil {
		return err
	}

	matched, modified, err := s.repo.HideCommentByUser(ctx, topicID, commentOID, userIDString(userID))
	if err != nil {
		return s.deleteCommentFallback(ctx, topicID, commentID, bizerr.InternalWrap("删除评论失败", err))
	}
	if !matched {
		return ErrCommentNotFound
	}
	if !modified {
		return ErrCommentDeleteFailed
	}

	if err := s.decrementCommentCounters(ctx, topicID, comment.RootCmtID); err != nil {
		return s.deleteCommentFallback(ctx, topicID, commentID, err)
	}
	return nil
}

func (s *Service) DeleteCommentAdmin(ctx context.Context, topicID, commentID string) error {
	comment, err := s.getCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	commentOID, err := parseObjectID(commentID)
	if err != nil {
		return err
	}

	matched, modified, err := s.repo.HideCommentAdmin(ctx, topicID, commentOID)
	if err != nil {
		return s.deleteCommentFallback(ctx, topicID, commentID, bizerr.InternalWrap("鍒犻櫎璇勮澶辫触", err))
	}
	if !matched {
		return ErrCommentNotFound
	}
	if !modified {
		return ErrCommentDeleteFailed
	}

	if err := s.decrementCommentCounters(ctx, topicID, comment.RootCmtID); err != nil {
		return s.deleteCommentFallback(ctx, topicID, commentID, err)
	}
	return nil
}

func (s *Service) ListByTopic(
	ctx context.Context,
	topicID, rootID string,
	viewerUserID int64,
	page, size int,
) (*PageResult[Comment], error) {
	page, size = pageSize(page, size, s.defaultPageSize())
	if err := s.ensureTopicExists(ctx, topicID); err != nil {
		return nil, err
	}
	if rootID == "" {
		rootID = DefaultRootCommentID
	}
	if rootID != DefaultRootCommentID {
		if _, err := parseObjectID(rootID); err != nil {
			return nil, ErrInvalidRootID
		}
	}

	filter := bson.M{
		"topicId":   topicID,
		"rootCmtId": rootID,
		"hasCheck":  bson.M{"$ne": false},
	}
	sort := bson.D{{Key: "_id", Value: 1}}
	if rootID == DefaultRootCommentID {
		sort = bson.D{{Key: "commentNum", Value: -1}, {Key: "_id", Value: -1}}
	}

	comments, total, err := s.repo.FindCommentsPage(ctx, filter, sort, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询评论列表失败", err)
	}

	if err := s.fillCommentLikeState(ctx, userIDString(viewerUserID), comments); err != nil {
		return nil, err
	}
	return NewPageResult(comments, total, page, size), nil
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, size int) (*PageResult[MyCommentItem], error) {
	page, size = pageSize(page, size, s.defaultPageSize())
	filter := bson.M{"user.userId": userIDString(userID), "hasCheck": bson.M{"$ne": false}}

	comments, total, err := s.repo.FindCommentsPage(
		ctx,
		filter,
		bson.D{{Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "_id", Value: -1}},
		page,
		size,
	)
	if err != nil {
		return nil, bizerr.InternalWrap("查询我的评论失败", err)
	}

	if err := s.fillCommentLikeState(ctx, userIDString(userID), comments); err != nil {
		return nil, err
	}

	items := make([]MyCommentItem, 0, len(comments))
	for _, comment := range comments {
		topic, err := s.getTopic(ctx, comment.TopicID, true)
		if err != nil {
			return nil, err
		}
		if topic == nil {
			return nil, ErrTopicNotFound
		}

		items = append(items, MyCommentItem{
			Comment: comment,
			Topic:   *topic,
		})
	}
	return NewPageResult(items, total, page, size), nil
}

func (s *Service) ListTargetUserComments(ctx context.Context, targetUserID int64, page, size int) (*PageResult[Comment], error) {
	page, size = pageSize(page, size, s.defaultPageSize())
	if targetUserID <= 0 {
		return nil, ErrTargetUserIDRequired
	}

	comments, total, err := s.repo.FindCommentsPage(
		ctx,
		bson.M{"user.userId": userIDString(targetUserID), "hasCheck": bson.M{"$ne": false}},
		bson.D{{Key: "_id", Value: -1}},
		page,
		size,
	)
	if err != nil {
		return nil, bizerr.InternalWrap("查询目标用户评论失败", err)
	}
	return NewPageResult(comments, total, page, size), nil
}

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func (s *Service) decrementCommentCounters(ctx context.Context, topicID, rootCommentID string) error {
	topicOID, err := parseObjectID(topicID)
	if err != nil {
		return err
	}

	ok, err := s.repo.IncrementTopicCommentNum(ctx, topicOID, -1)
	if err != nil {
		return bizerr.InternalWrap("删除评论失败", err)
	}
	if !ok {
		return ErrCommentDeleteFailed
	}

	if rootCommentID == "" || rootCommentID == DefaultRootCommentID {
		return nil
	}

	rootOID, err := parseObjectID(rootCommentID)
	if err != nil {
		return err
	}

	ok, err = s.repo.IncrementRootCommentNum(ctx, rootOID, -1)
	if err != nil {
		return bizerr.InternalWrap("删除评论失败", err)
	}
	if !ok {
		return ErrCommentDeleteFailed
	}
	return nil
}

func (s *Service) deleteCommentFallback(ctx context.Context, topicID, commentID string, cause error) error {
	if s.producer != nil {
		if err := s.producer.SendDeleteComment(ctx, topicID, commentID); err != nil {
			s.logger.Warn("send delete comment mq failed", zap.Error(err), zap.String("commentID", commentID))
		}
	}
	return cause
}
