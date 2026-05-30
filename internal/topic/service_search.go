package topic

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) Search(
	ctx context.Context,
	userID string,
	accountType string,
	themeIDs []string,
	content string,
	page, size, ordCreated int,
) (*PageResult[Topic], error) {
	if ordCreated == 0 {
		return s.searchHot(ctx, userID, accountType, themeIDs, content, page, size)
	}

	filter := bson.M{"hasCheck": true}
	if len(themeIDs) > 0 {
		filter["themeId"] = bson.M{"$in": themeIDs}
	}
	if strings.TrimSpace(content) != "" {
		filter["content"] = primitive.Regex{Pattern: ".*" + content + ".*", Options: "i"}
	}
	return s.listByFilter(ctx, filter, page, size, userID, accountType, bson.D{{Key: "_id", Value: -1}, {Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "visitedNum", Value: -1}})
}

func (s *Service) searchHot(
	ctx context.Context,
	userID string,
	accountType string,
	themeIDs []string,
	content string,
	page, size int,
) (*PageResult[Topic], error) {
	match := bson.M{"hasCheck": true}
	if len(themeIDs) > 0 {
		match["themeId"] = bson.M{"$in": themeIDs}
	}
	if strings.TrimSpace(content) != "" {
		match["content"] = primitive.Regex{Pattern: ".*" + content + ".*", Options: "i"}
	}

	topics, total, err := s.repo.SearchHotTopics(ctx, match, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询热门帖子失败", err)
	}

	s.prepareTopics(topics)
	if err := s.fillTopicUserCertification(ctx, topics); err != nil {
		return nil, err
	}
	if err := s.fillLikeAndCollection(ctx, userID, accountType, topics); err != nil {
		s.logger.Warn("fill topic like/collection failed", zap.Error(err))
	}
	return NewPageResult(topics, total, page, size), nil
}
