package topic

import (
	"context"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

const (
	topicAccountTypeBase      = "base"
	topicAccountTypeAnonymous = "anonymous"
)

func (s *Service) ensureThemeExists(ctx context.Context, themeID string) error {
	if strings.TrimSpace(themeID) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	exists, err := s.repo.ThemeExists(ctx, themeID)
	if err != nil {
		return bizerr.InternalWrap("查询主题失败", err)
	}
	if !exists {
		return ErrThemeNotFound
	}
	return nil
}

func (s *Service) resolveThemeName(ctx context.Context, themeID string) string {
	name, err := s.repo.FindThemeName(ctx, themeID)
	if err != nil || strings.TrimSpace(name) == "" {
		return themeID
	}
	return name
}

func (s *Service) resolveTopicAuthor(ctx context.Context, claims *jwtutil.Claims, accountType string) (*topicAuthor, error) {
	if err := validateTopicClaims(claims); err != nil {
		return nil, err
	}

	accountType, err := normalizeTopicAccountType(accountType)
	if err != nil {
		return nil, err
	}

	if accountType == topicAccountTypeAnonymous {
		author, err := s.repo.FindUserByRootAndAccountType(ctx, claims.RootUserID, topicAccountTypeAnonymous)
		if err != nil {
			return nil, bizerr.InternalWrap("查询匿名身份失败", err)
		}
		if author == nil {
			return nil, ErrAnonymousAccountNotFound
		}
		return author, nil
	}

	author, err := s.repo.FindUserByID(ctx, claims.RootUserID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询用户失败", err)
	}
	if author == nil {
		return nil, ErrUserNotFound
	}
	return author, nil
}

func (s *Service) getTopicByID(ctx context.Context, topicID string, onlyChecked bool) (*Topic, error) {
	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return nil, err
	}

	topic, err := s.repo.FindTopicByID(ctx, oid, onlyChecked)
	if err != nil {
		return nil, bizerr.InternalWrap("查询帖子失败", err)
	}
	if topic != nil {
		s.prepareTopic(topic)
	}
	return topic, nil
}

func (s *Service) prepareTopic(topic *Topic) {
	if topic == nil {
		return
	}
	topic.Imgs = ensureSlice(topic.Imgs)
	topic.CreatedTime = topic.ID.Timestamp().Format("2006-01-02 15:04:05")
}

func (s *Service) prepareTopics(topics []Topic) {
	for i := range topics {
		s.prepareTopic(&topics[i])
	}
}

func (s *Service) fillLikeAndCollection(ctx context.Context, userID string, topics []Topic) error {
	if userID == "" || len(topics) == 0 {
		return nil
	}

	indexes := make(map[string]int, len(topics))
	for i := range topics {
		indexes[topics[i].ID.Hex()] = i
		topics[i].HasLike = false
		topics[i].HasCollection = false
	}

	likeDocs, err := s.repo.FindTopicStateDocs(ctx, mongoCollTopicLike, userID)
	if err != nil {
		return err
	}
	applyTopicFlags(indexes, topics, likeDocs, true)

	collectionDocs, err := s.repo.FindTopicStateDocs(ctx, mongoCollTopicCollection, userID)
	if err != nil {
		return err
	}
	applyTopicFlags(indexes, topics, collectionDocs, false)
	return nil
}

func applyTopicFlags(indexes map[string]int, topics []Topic, docs []topicStateDoc, isLike bool) {
	for _, doc := range docs {
		for _, id := range doc.TopicIDs {
			idx, ok := indexes[id]
			if !ok {
				continue
			}
			if isLike {
				topics[idx].HasLike = true
				continue
			}
			topics[idx].HasCollection = true
		}
	}
}

func parseTopicObjectID(topicID string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(topicID))
	if err != nil {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}
	return oid, nil
}

func userIDString(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

func validateTopicClaims(claims *jwtutil.Claims) error {
	if claims == nil || claims.UserID <= 0 || claims.RootUserID <= 0 {
		return ErrInvalidAuthClaims
	}
	return nil
}

func normalizeTopicAccountType(accountType string) (string, error) {
	accountType = strings.TrimSpace(accountType)
	switch accountType {
	case topicAccountTypeBase, topicAccountTypeAnonymous:
		return accountType, nil
	default:
		return "", bizerr.Param(errMsgInvalidParam)
	}
}
