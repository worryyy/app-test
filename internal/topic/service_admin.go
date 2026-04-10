package topic

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

func (s *Service) ListAdmin(ctx context.Context, page, size int) (*PageResult[Topic], error) {
	topics, total, err := s.repo.FindTopicsPage(
		ctx,
		bson.M{},
		bson.D{{Key: "_id", Value: -1}},
		page,
		size,
	)
	if err != nil {
		return nil, bizerr.InternalWrap("鏌ヨ甯栧瓙鍒楄〃澶辫触", err)
	}

	s.prepareTopics(topics)
	return NewPageResult(topics, total, page, size), nil
}

func (s *Service) CreateAdmin(ctx context.Context, claims *jwtutil.Claims, req *CreateTopicReq) (*Topic, error) {
	if claims == nil {
		return nil, ErrInvalidAuthClaims
	}
	if req == nil {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if err := s.ensureThemeExists(ctx, req.ThemeID); err != nil {
		return nil, err
	}

	author, err := s.resolveTopicAuthor(ctx, claims, req.AccountType)
	if err != nil {
		return nil, err
	}

	topic := buildAdminTopic(author, req)
	if _, err := s.repo.CreateTopic(ctx, topic); err != nil {
		return nil, bizerr.InternalWrap("鍒涘缓甯栧瓙澶辫触", err)
	}

	s.prepareTopic(topic)
	return topic, nil
}

func (s *Service) UpdateAdmin(ctx context.Context, topicID string, req *UpdateTopicReq) error {
	if req == nil {
		return bizerr.Param(errMsgInvalidParam)
	}

	update := buildAdminTopicUpdate(req)
	if len(update) == 0 {
		return nil
	}

	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return err
	}

	ok, err := s.repo.UpdateTopicAdmin(ctx, oid, update)
	if err != nil {
		return bizerr.InternalWrap("鏇存柊甯栧瓙澶辫触", err)
	}
	if !ok {
		return ErrTopicNotFound
	}
	return nil
}

func (s *Service) DeleteAdmin(ctx context.Context, topicID string) error {
	oid, err := parseTopicObjectID(topicID)
	if err != nil {
		return err
	}

	ok, err := s.repo.HideTopic(ctx, oid, "", true)
	if err != nil {
		return bizerr.InternalWrap("鍒犻櫎甯栧瓙澶辫触", err)
	}
	if !ok {
		return ErrTopicNotFound
	}

	if err := s.repo.CleanupDeletedTopic(ctx, topicID); err != nil {
		return bizerr.InternalWrap("鍒犻櫎甯栧瓙澶辫触", err)
	}
	return nil
}

func buildAdminTopic(author *topicAuthor, req *CreateTopicReq) *Topic {
	title := " "
	if req != nil && req.Title != "" {
		title = req.Title
	}

	return &Topic{
		ThemeID:       req.ThemeID,
		UserID:        userIDString(author.ID),
		Title:         title,
		Content:       req.Content,
		Imgs:          ensureSlice(req.Imgs),
		HasCheck:      true,
		VisitedNum:    0,
		LikeNum:       0,
		CommentNum:    0,
		CollectionNum: 0,
		Ext:           req.Ext,
		AccountType:   author.AccountType,
		NickName:      author.Nickname,
		Avatar:        author.Avatar,
		HasLike:       false,
		HasCollection: false,
	}
}

func buildAdminTopicUpdate(req *UpdateTopicReq) bson.M {
	update := bson.M{}
	if req == nil {
		return update
	}
	if req.Title != "" {
		update["title"] = req.Title
	}
	if req.Content != "" {
		update["content"] = req.Content
	}
	if len(req.Imgs) > 0 {
		update["imgs"] = ensureSlice(req.Imgs)
	}
	if req.Ext != nil && *req.Ext != "" {
		update["ext"] = *req.Ext
	}
	return update
}
