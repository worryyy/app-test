package chat

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	conversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("查询会话列表失败", err)
	}
	if len(conversationIDs) == 0 {
		return []Conversation{}, nil
	}

	conversations, err := s.repo.FindConversationsByIDs(ctx, conversationIDs)
	if err != nil {
		return nil, bizerr.InternalWrap("查询会话列表失败", err)
	}
	return conversations, nil
}

func (s *Service) EnterConversation(ctx context.Context, userID int64, req ConversationEnterReq) error {
	if userID <= 0 || strings.TrimSpace(req.ConversationID) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	member, err := s.repo.FindConversationMember(ctx, req.ConversationID, userIDString(userID))
	if err != nil {
		return bizerr.InternalWrap("查询会话成员失败", err)
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
		return bizerr.InternalWrap("更新会话失败", err)
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
		return nil, bizerr.InternalWrap("查询未读数失败", err)
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
		return nil, bizerr.InternalWrap("查询会话失败", err)
	}
	userConversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("查询会话失败", err)
	}

	resultIDs := make([]string, 0)
	for _, conversationID := range targetConversationIDs {
		if slices.Contains(userConversationIDs, conversationID) {
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
		return "", bizerr.InternalWrap("查询聊天对象失败", err)
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
		return bizerr.InternalWrap("删除会话失败", err)
	}
}
