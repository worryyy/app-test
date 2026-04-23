package agentchat

import (
	"context"
	"strconv"
	"strings"

	agentv1 "github.com/Milchstrassse/Ecampus-go/internal/agentchat/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

func (s *Service) ListConversations(ctx context.Context, rootUserID int64, page, size int) (*pagination.PageResult[Conversation], error) {
	page, size = normalizeConversationPage(page, size)
	items, total, err := s.listConversations(ctx, rootUserID, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询 agent 会话失败", err)
	}
	return pagination.NewPageResult(items, total, page, size), nil
}

func (s *Service) GetHistory(
	ctx context.Context,
	rootUserID int64,
	conversationID string,
	beforeSequenceNo *int64,
	size int,
) (*HistoryResponse, error) {
	conversation, err := s.requireOwnedConversation(ctx, conversationID, rootUserID)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	resp, err := s.client.GetSessionHistory(callCtx, buildHistoryRequest(rootUserID, conversation.SessionID, beforeSequenceNo, size))
	if err != nil {
		return nil, s.mapCallError(err)
	}
	return toHistoryResponse(conversation.SessionID, resp), nil
}

func (s *Service) DeleteConversation(ctx context.Context, rootUserID int64, conversationID string) error {
	conversation, err := s.requireOwnedConversation(ctx, conversationID, rootUserID)
	if err != nil {
		return err
	}
	if err := s.deleteRemoteConversation(ctx, rootUserID, conversation.SessionID); err != nil {
		return err
	}

	deleted, err := s.deleteConversationRecord(ctx, conversation.SessionID, rootUserID)
	if err != nil {
		return bizerr.InternalWrap("删除 agent 会话失败", err)
	}
	if !deleted {
		return ErrConversationNotFound
	}
	return nil
}

func (s *Service) resolveConversation(ctx context.Context, input TurnInput) (*Conversation, error) {
	conversationID := strings.TrimSpace(input.ConversationID)
	if conversationID == "" {
		return newConversation(input), nil
	}
	return s.requireOwnedConversation(ctx, conversationID, input.RootUserID)
}

func (s *Service) requireOwnedConversation(ctx context.Context, conversationID string, rootUserID int64) (*Conversation, error) {
	conversation, err := s.getConversation(ctx, strings.TrimSpace(conversationID))
	if err != nil {
		return nil, bizerr.InternalWrap("查询 agent 会话失败", err)
	}
	if conversation == nil {
		return nil, ErrConversationNotFound
	}
	if conversation.RootUserID != rootUserID {
		return nil, ErrConversationAccessDenied
	}
	return conversation, nil
}

func (s *Service) deleteRemoteConversation(ctx context.Context, rootUserID int64, sessionID string) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	_, err := s.client.DeleteSession(callCtx, &agentv1.DeleteSessionRequest{
		UserId:    strconv.FormatInt(rootUserID, 10),
		SessionId: sessionID,
	})
	if err == nil || status.Code(err) == codes.NotFound {
		return nil
	}
	if status.Code(err) == codes.PermissionDenied {
		return ErrConversationDeleteDenied
	}
	return s.mapCallError(err)
}

func buildHistoryRequest(rootUserID int64, sessionID string, beforeSequenceNo *int64, size int) *agentv1.GetSessionHistoryRequest {
	req := &agentv1.GetSessionHistoryRequest{
		UserId:    strconv.FormatInt(rootUserID, 10),
		SessionId: sessionID,
		Limit:     int32(normalizeHistoryLimit(size)),
	}
	if beforeSequenceNo != nil {
		req.BeforeSequenceNo = *beforeSequenceNo
	}
	return req
}

func normalizeHistoryLimit(size int) int {
	if size <= 0 {
		return defaultHistoryPageSize
	}
	if size > maxHistoryPageSize {
		return maxHistoryPageSize
	}
	return size
}

func toHistoryResponse(conversationID string, resp *agentv1.GetSessionHistoryResponse) *HistoryResponse {
	result := &HistoryResponse{
		ConversationID: conversationID,
		Turns:          []HistoryTurn{},
	}
	if resp == nil {
		return result
	}

	result.Turns = make([]HistoryTurn, 0, len(resp.GetTurns()))
	for _, turn := range resp.GetTurns() {
		result.Turns = append(result.Turns, HistoryTurn{
			SequenceNo: turn.GetSequenceNo(),
			Role:       turn.GetRole(),
			Content:    turn.GetContent(),
			CreatedAt:  turn.GetCreatedAt(),
			Domain:     enumValue(turn.GetDomain().String(), "DOMAIN_"),
		})
	}
	result.HasMore = resp.GetHasMore()
	result.NextBeforeSequenceNo = resp.GetNextBeforeSequenceNo()
	return result
}

func newConversation(input TurnInput) *Conversation {
	return &Conversation{
		SessionID:       "agent-session-" + snowflake.Generate().String(),
		RootUserID:      input.RootUserID,
		CreatorUserID:   input.CurrentUserID,
		LastActorUserID: input.CurrentUserID,
		Title:           buildConversationTitle(input.Content),
		Status:          conversationStatusPending,
	}
}
