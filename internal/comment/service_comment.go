package comment

import (
	"context"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

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
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, bizerr.InternalWrap("query user failed", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *Service) loadUserByStringID(ctx context.Context, userID string) (*userRecord, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	if err != nil {
		return nil, bizerr.Internal("invalid topic author data")
	}
	return s.loadUser(ctx, id)
}

func (s *Service) getTopic(ctx context.Context, topicID string, onlyChecked bool) (*CommentTopic, error) {
	oid, err := parseCommentObjectID(topicID)
	if err != nil {
		return nil, err
	}

	topic, err := s.repo.FindTopicByID(ctx, oid, onlyChecked)
	if err != nil {
		return nil, bizerr.InternalWrap("query topic failed", err)
	}
	if topic == nil {
		return nil, nil
	}

	if topic.Imgs == nil {
		topic.Imgs = []string{}
	}
	createdAt := topic.ID.Timestamp()
	topic.CreatedTime = &createdAt
	return topic, nil
}

func (s *Service) ensureTopicExists(ctx context.Context, topicID string) error {
	topic, err := s.getTopic(ctx, topicID, false)
	if err != nil {
		return err
	}
	if topic == nil {
		return ErrTopicNotFound
	}
	return nil
}

func (s *Service) getCommentByID(ctx context.Context, commentID string) (*Comment, error) {
	oid, err := parseCommentObjectID(commentID)
	if err != nil {
		return nil, err
	}

	cmt, err := s.repo.FindCommentByID(ctx, oid)
	if err != nil {
		return nil, bizerr.InternalWrap("query comment failed", err)
	}
	if cmt == nil {
		return nil, ErrCommentNotFound
	}
	return cmt, nil
}

func (s *Service) getHasLikeBatch(ctx context.Context, userID string, comments []Comment) (map[string]struct{}, error) {
	ids := make([]string, 0, len(comments))
	for _, comment := range comments {
		if !comment.ID.IsZero() {
			ids = append(ids, comment.ID.Hex())
		}
	}

	liked, err := s.repo.FindLikedCommentIDs(ctx, ids, userID)
	if err != nil {
		return nil, bizerr.InternalWrap("query comment like status failed", err)
	}
	return liked, nil
}

func (s *Service) fillCommentLikeState(ctx context.Context, userID string, comments []Comment) error {
	if strings.TrimSpace(userID) == "" || len(comments) == 0 {
		return nil
	}

	liked, err := s.getHasLikeBatch(ctx, userID, comments)
	if err != nil {
		return err
	}
	for i := range comments {
		_, ok := liked[comments[i].ID.Hex()]
		comments[i].HasLike = ok
	}
	return nil
}

func buildCommentUser(user *userRecord) CommentUser {
	if user == nil {
		return CommentUser{}
	}
	return CommentUser{
		UserID:      strconv.FormatInt(user.ID, 10),
		Avatar:      user.Avatar,
		NickName:    user.Nickname,
		AccountType: user.AccountType,
		Signature:   user.Signature,
	}
}

func normalizeCommentAccountType(accountType string) string {
	switch strings.ToLower(strings.TrimSpace(accountType)) {
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
		return ErrAnonymousCommentForbidden
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
		return ErrCommentSelfRolePlayForbidden
	}
	return nil
}

func userIDString(userID int64) string {
	if userID <= 0 {
		return ""
	}
	return strconv.FormatInt(userID, 10)
}

func parseCommentObjectID(raw string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(raw))
	if err != nil {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}
	return oid, nil
}
