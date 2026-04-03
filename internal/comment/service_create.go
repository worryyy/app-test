package comment

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

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

	comment := Comment{
		TopicID:     topicID,
		Comment:     content,
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

func (s *Service) DeleteComment(ctx context.Context, topicID, commentID string, userID int64, isAdmin bool) error {
	comment, err := s.getCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	commentOID, err := parseCommentObjectID(commentID)
	if err != nil {
		return err
	}

	matched, modified, err := s.repo.HideComment(ctx, topicID, commentOID, userIDString(userID), isAdmin)
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

func (s *Service) decrementCommentCounters(ctx context.Context, topicID, rootCommentID string) error {
	topicOID, err := parseCommentObjectID(topicID)
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

	rootOID, err := parseCommentObjectID(rootCommentID)
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
