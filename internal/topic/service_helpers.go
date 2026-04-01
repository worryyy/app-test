package topic

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type topicAuthor struct {
	ID          int64  `gorm:"column:id"`
	Nickname    string `gorm:"column:nickname"`
	Avatar      string `gorm:"column:avatar"`
	AccountType string `gorm:"column:account_type"`
	RootUserID  int64  `gorm:"column:root_user_id"`
}

func (topicAuthor) TableName() string {
	return "campus_user"
}

type campusThemeID struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Name    string             `bson:"name"`
	ThemeID string             `bson:"themeId"`
}

type topicStateDoc struct {
	TopicIDs []string `bson:"topicIds"`
}

func (s *Service) ensureThemeExists(ctx context.Context, themeID string) error {
	if strings.TrimSpace(themeID) == "" {
		return result.ErrParam
	}

	err := s.mongoDB.Collection("campus_theme_id").FindOne(ctx, bson.M{"themeId": themeID}).Err()
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return result.NewBizError(result.CodeNotExisted, "themeId is not existed")
	}
	return fmt.Errorf("check theme exists: %w", err)
}

func (s *Service) resolveThemeName(ctx context.Context, themeID string) string {
	var doc campusThemeID
	err := s.mongoDB.Collection("campus_theme_id").FindOne(ctx, bson.M{"themeId": themeID}).Decode(&doc)
	if err != nil {
		return themeID
	}
	return doc.Name
}

func (s *Service) resolveTopicAuthor(ctx context.Context, claims *jwtutil.Claims, accountType string) (*topicAuthor, error) {
	if claims == nil {
		return nil, result.ErrParam
	}

	targetUserID := claims.UserID
	targetAccountType := strings.TrimSpace(accountType)
	if targetAccountType == "" {
		targetAccountType = claims.AccountType
	}

	if targetAccountType != "" && targetAccountType != claims.AccountType {
		var alt topicAuthor
		err := s.db.WithContext(ctx).
			Where(colTopicRootUserID+" = ? AND "+colTopicAccountType+" = ?", claims.RootUserID, targetAccountType).
			First(&alt).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, result.NewBizError(result.CodeFail, "匿名账号不存在")
		}
		if err != nil {
			return nil, fmt.Errorf("query topic author by account type: %w", err)
		}
		return &alt, nil
	}

	var author topicAuthor
	err := s.db.WithContext(ctx).Where(colTopicUserID+" = ?", targetUserID).First(&author).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, result.NewBizError(result.CodeNotExisted, "用户不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("query topic author: %w", err)
	}
	return &author, nil
}

func (s *Service) getTopicByID(ctx context.Context, topicID string, onlyChecked bool) (*Topic, error) {
	oid, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return nil, fmt.Errorf("invalid topic id: %w", err)
	}

	filter := bson.M{"_id": oid}
	if onlyChecked {
		filter["hasCheck"] = true
	}

	var topic Topic
	if err := s.topicColl().FindOne(ctx, filter).Decode(&topic); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic by id: %w", err)
	}

	s.prepareTopic(&topic)
	return &topic, nil
}

func (s *Service) prepareTopic(topic *Topic) {
	if topic == nil {
		return
	}
	topic.Imgs = result.EnsureSlice(topic.Imgs)
	createdAt := topic.ID.Timestamp()
	topic.CreatedTime = &createdAt
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

	ids := make(map[string]int, len(topics))
	for i := range topics {
		ids[topics[i].ID.Hex()] = i
		topics[i].HasLike = false
		topics[i].HasCollection = false
	}

	if err := s.fillTopicFlags(ctx, "campus_topic_like", userID, ids, topics, true); err != nil {
		return err
	}
	if err := s.fillTopicFlags(ctx, "campus_topic_collection", userID, ids, topics, false); err != nil {
		return err
	}
	return nil
}

func (s *Service) fillTopicFlags(
	ctx context.Context,
	collName string,
	userID string,
	indexes map[string]int,
	topics []Topic,
	isLike bool,
) error {
	cur, err := s.mongoDB.Collection(collName).Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return fmt.Errorf("load %s docs: %w", collName, err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var docs []topicStateDoc
	if err := cur.All(ctx, &docs); err != nil {
		return fmt.Errorf("decode %s docs: %w", collName, err)
	}

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
	return nil
}

func userIDString(userID int64) string {
	return strconv.FormatInt(userID, 10)
}
