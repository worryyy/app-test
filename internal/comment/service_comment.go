package comment

import (
	"context"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

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
	oid, err := parseObjectID(topicID)
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
	createdAt := topic.ID.Timestamp().Local()
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
	oid, err := parseObjectID(commentID)
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

func (s *Service) ListCommentAdmin(ctx context.Context, topicID string, page, size int) (*PageResult[Comment], error) {
	page, size = pageSize(page, size, s.defaultPageSize())
	if err := s.ensureTopicExists(ctx, topicID); err != nil {
		return nil, err
	}

	comments, total, err := s.repo.FindCommentsPage(
		ctx,
		bson.M{"topicId": topicID},
		bson.D{{Key: "_id", Value: -1}},
		page,
		size,
	)
	if err != nil {
		return nil, bizerr.InternalWrap("query admin comment list failed", err)
	}
	if err := s.fillCommentUserCertification(ctx, comments); err != nil {
		return nil, err
	}
	return NewPageResult(comments, total, page, size), nil
}
func buildCommentUser(user *userRecord) CommentUser {
	if user == nil {
		return CommentUser{}
	}
	return CommentUser{
		UserID:               strconv.FormatInt(user.ID, 10),
		Avatar:               user.Avatar,
		NickName:             user.Nickname,
		AccountType:          user.AccountType,
		Signature:            user.Signature,
		StuIsCheck:           boolPtr(user.StuIsCheck),
		ProvisionalExpiresAt: user.ProvisionalExpiresAt,
	}
}

func (s *Service) fillCommentUserCertification(ctx context.Context, comments []Comment) error {
	userIDs := collectCommentUserIDs(comments)
	if len(userIDs) == 0 {
		return nil
	}

	users, err := s.repo.FindUsersByIDs(ctx, userIDs)
	if err != nil {
		return bizerr.InternalWrap("query comment user certification failed", err)
	}
	applyCommentUserCertification(comments, users)
	return nil
}

func collectCommentUserIDs(comments []Comment) []int64 {
	seen := make(map[int64]struct{}, len(comments))
	ids := make([]int64, 0, len(comments))
	for _, comment := range comments {
		ids = appendCommentUserID(ids, seen, comment.User.UserID)
		if comment.Parent != nil {
			ids = appendCommentUserID(ids, seen, comment.Parent.UserID)
		}
	}
	return ids
}

func appendCommentUserID(ids []int64, seen map[int64]struct{}, raw string) []int64 {
	id, ok := parseCommentUserID(raw)
	if !ok {
		return ids
	}
	if _, exists := seen[id]; exists {
		return ids
	}
	seen[id] = struct{}{}
	return append(ids, id)
}

func applyCommentUserCertification(comments []Comment, users map[int64]userRecord) {
	for i := range comments {
		applyCommentUserCertificationToUser(&comments[i].User, users)
		if comments[i].Parent != nil {
			applyCommentUserCertificationToUser(comments[i].Parent, users)
		}
	}
}

func applyCommentUserCertificationToUser(commentUser *CommentUser, users map[int64]userRecord) {
	if commentUser == nil {
		return
	}
	id, ok := parseCommentUserID(commentUser.UserID)
	if !ok {
		return
	}
	user, ok := users[id]
	if !ok {
		return
	}
	commentUser.StuIsCheck = boolPtr(user.StuIsCheck)
	commentUser.ProvisionalExpiresAt = user.ProvisionalExpiresAt
}

func parseCommentUserID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func boolPtr(value bool) *bool {
	return &value
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

func parseObjectID(raw string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(raw))
	if err != nil {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}
	return oid, nil
}
