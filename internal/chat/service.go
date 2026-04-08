package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const (
	maxPageSize              = 100
	historyMessageFetchLimit = 50
)

type Service struct {
	repo   *Repository
	redis  *redis.Client
	cfg    *config.Config
	logger *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:   NewRepository(db, mongoDB),
		redis:  rds,
		cfg:    cfg,
		logger: logger,
	}
}

func (s *Service) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	conversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("query conversations failed", err)
	}
	if len(conversationIDs) == 0 {
		return []Conversation{}, nil
	}

	conversations, err := s.repo.FindConversationsByIDs(ctx, conversationIDs)
	if err != nil {
		return nil, bizerr.InternalWrap("query conversations failed", err)
	}
	return conversations, nil
}

func (s *Service) EnterConversation(ctx context.Context, userID int64, req ConversationEnterReq) error {
	if userID <= 0 || strings.TrimSpace(req.ConversationID) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	member, err := s.repo.FindConversationMember(ctx, req.ConversationID, userIDString(userID))
	if err != nil {
		return bizerr.InternalWrap("query conversation member failed", err)
	}
	if member == nil {
		return ErrConversationNotFound
	}
	if member.UnreadCount == 0 {
		return nil
	}

	if strings.TrimSpace(req.LastMessageID) == "" {
		return ErrLastMessageIDRequired
	}
	lastMessageID, err := strconv.ParseInt(strings.TrimSpace(req.LastMessageID), 10, 64)
	if err != nil || lastMessageID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	ok, err := s.repo.UpdateConversationMemberReadState(ctx, req.ConversationID, userIDString(userID), lastMessageID)
	if err != nil {
		return bizerr.InternalWrap("update conversation failed", err)
	}
	if !ok {
		return ErrConversationUpdateFailed
	}
	return nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID int64, conversationID string) ([]ConversationUnreadCount, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}

	list, err := s.repo.FindConversationUnreadCounts(ctx, conversationID, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("query unread count failed", err)
	}
	return list, nil
}

func (s *Service) QueryConversation(ctx context.Context, userID int64, targetUserID string) ([]string, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(targetUserID) == "" {
		return nil, ErrTargetUserIDRequired
	}

	targetConversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, strings.TrimSpace(targetUserID))
	if err != nil {
		return nil, bizerr.InternalWrap("query conversation failed", err)
	}
	userConversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("query conversation failed", err)
	}

	userConversationSet := make(map[string]struct{}, len(userConversationIDs))
	for _, conversationID := range userConversationIDs {
		userConversationSet[conversationID] = struct{}{}
	}

	resultIDs := make([]string, 0)
	for _, conversationID := range targetConversationIDs {
		if _, ok := userConversationSet[conversationID]; ok {
			resultIDs = append(resultIDs, conversationID)
		}
	}
	return resultIDs, nil
}

func (s *Service) GetPeerUserID(ctx context.Context, conversationID string, currentUserID int64) (string, error) {
	if currentUserID <= 0 {
		return "", bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(conversationID) == "" {
		return "", ErrConversationIDRequired
	}

	member, err := s.repo.FindPeerConversationMember(ctx, conversationID, userIDString(currentUserID))
	if err != nil {
		return "", bizerr.InternalWrap("query peer user failed", err)
	}
	if member == nil {
		return "", ErrConversationPeerNotFound
	}
	return member.UserID, nil
}

func (s *Service) DeleteConversation(ctx context.Context, userID int64, conversationID string) error {
	if userID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(conversationID) == "" {
		return ErrConversationIDRequired
	}

	err := s.repo.DeleteConversationForUser(ctx, conversationID, userIDString(userID))
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errRepoConversationNotFound):
		return ErrConversationNotFound
	case errors.Is(err, errRepoConversationMemberMiss):
		return ErrConversationDeleteDenied
	case errors.Is(err, errRepoConversationDeleteFailed):
		return ErrConversationDeleteFailed
	default:
		return bizerr.InternalWrap("delete conversation failed", err)
	}
}

func (s *Service) GetOfflineMessages(ctx context.Context, userID int64, lastMessageID int64) ([]Message, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	conversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("query offline messages failed", err)
	}
	if len(conversationIDs) == 0 {
		return []Message{}, nil
	}

	messages, err := s.repo.FindMessagesAfter(ctx, conversationIDs, lastMessageID)
	if err != nil {
		return nil, bizerr.InternalWrap("query offline messages failed", err)
	}
	return messages, nil
}

func (s *Service) GetHistoryMessages(
	ctx context.Context,
	userID int64,
	conversationID string,
	oldestMessageID *int64,
	page, size int,
) (*PageResult[Message], error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}
	page, size = normalizePage(page, size, s.defaultPageSize())
	if size > historyMessageFetchLimit {
		size = historyMessageFetchLimit
	}

	member, err := s.repo.FindConversationMember(ctx, conversationID, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("query history messages failed", err)
	}
	if member == nil {
		return nil, ErrConversationAccessDenied
	}

	messages, total, err := s.repo.FindConversationMessagesBefore(ctx, conversationID, oldestMessageID, int64(size))
	if err != nil {
		return nil, bizerr.InternalWrap("query history messages failed", err)
	}
	return NewPageResult(messages, total, page, size), nil
}

func (s *Service) HasUnreadMessages(ctx context.Context, userID int64) (bool, error) {
	if userID <= 0 {
		return false, bizerr.Param(errMsgInvalidParam)
	}

	total, err := s.repo.SumUnreadCount(ctx, userIDString(userID))
	if err != nil {
		return false, bizerr.InternalWrap("query unread messages failed", err)
	}
	return total != 0, nil
}

func (s *Service) HandleWSMessage(ctx context.Context, senderID int64, payload []byte) (*Message, error) {
	body := make(map[string]any)
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, ErrMessageParseFailed
	}

	handleType := strings.ToUpper(strings.TrimSpace(stringField(body, "handleType", "handle_type")))
	if handleType == "INIT" {
		return s.handleInitMessage(ctx, userIDString(senderID), body)
	}
	return s.handleChatMessage(ctx, userIDString(senderID), body)
}

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func normalizePage(page, size, defaultSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

func userIDString(userID int64) string {
	return strconv.FormatInt(userID, 10)
}
