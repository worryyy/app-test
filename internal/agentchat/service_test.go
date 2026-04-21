package agentchat

import (
	"context"
	"testing"

	agentv1 "github.com/Milchstrassse/Ecampus-go/internal/agentchat/agentv1"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

type fakeAgentClient struct {
	handleTurnResp *agentv1.HandleTurnResponse
	handleTurnErr  error
	historyResp    *agentv1.GetSessionHistoryResponse
	historyErr     error
	deleteResp     *agentv1.DeleteSessionResponse
	deleteErr      error
}

func (f fakeAgentClient) HandleTurn(_ context.Context, _ *agentv1.HandleTurnRequest) (*agentv1.HandleTurnResponse, error) {
	return f.handleTurnResp, f.handleTurnErr
}

func (f fakeAgentClient) StreamHandleTurn(_ context.Context, _ *agentv1.HandleTurnRequest) (agentv1.AgentService_StreamHandleTurnClient, error) {
	return nil, nil
}

func (f fakeAgentClient) GetSessionHistory(_ context.Context, _ *agentv1.GetSessionHistoryRequest) (*agentv1.GetSessionHistoryResponse, error) {
	return f.historyResp, f.historyErr
}

func (f fakeAgentClient) DeleteSession(_ context.Context, _ *agentv1.DeleteSessionRequest) (*agentv1.DeleteSessionResponse, error) {
	return f.deleteResp, f.deleteErr
}

func TestHandleTurnCreatesConversationAndPersistsResult(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db, nil, &config.Config{
		Agent: config.AgentConfig{Enabled: true, TimeoutMS: 1000},
	}, zap.NewNop(), fakeAgentClient{
		handleTurnResp: &agentv1.HandleTurnResponse{
			Domain:       agentv1.Domain_DOMAIN_CHAT,
			Intent:       agentv1.Intent_INTENT_CHAT_GENERAL,
			Mode:         agentv1.Mode_MODE_GENERATE,
			AnswerText:   "测试回复",
			TraceId:      "trace-1",
			ErrorCode:    agentv1.ErrorCode_ERROR_CODE_UNSPECIFIED,
			References:   []*agentv1.Reference{{Source: "kb", Ref: "doc#1", Score: 0.8}},
			CapabilityId: "chat.general",
		},
	})

	resp, err := svc.HandleTurn(context.Background(), TurnInput{
		Content:       "你好，帮我总结一下",
		RootUserID:    1001,
		CurrentUserID: 2002,
		AccountType:   "anonymous",
	})
	if err != nil {
		t.Fatalf("HandleTurn error: %v", err)
	}
	if resp.RequestID == "" {
		t.Fatalf("expected request id")
	}
	if resp.ConversationID == "" {
		t.Fatalf("expected conversation id")
	}
	if resp.Result.AnswerText != "测试回复" {
		t.Fatalf("answer = %q, want 测试回复", resp.Result.AnswerText)
	}
	if resp.Result.Domain != "chat" {
		t.Fatalf("domain = %q, want chat", resp.Result.Domain)
	}
	if len(resp.Result.References) != 1 {
		t.Fatalf("references len = %d, want 1", len(resp.Result.References))
	}

	conversation := findConversation(t, db, resp.ConversationID)
	if conversation == nil {
		t.Fatalf("expected conversation to be persisted")
	}
	if conversation.RootUserID != 1001 {
		t.Fatalf("root user id = %d, want 1001", conversation.RootUserID)
	}
	if conversation.CreatorUserID != 2002 {
		t.Fatalf("creator user id = %d, want 2002", conversation.CreatorUserID)
	}
	if conversation.Status != conversationStatusReady {
		t.Fatalf("status = %q, want ready", conversation.Status)
	}
	if conversation.LastAssistantPreview == "" {
		t.Fatalf("expected assistant preview")
	}
}

func TestGetHistoryChecksOwnershipAndMapsTurns(t *testing.T) {
	db := openTestDB(t)
	saveConversation(t, db, &Conversation{
		SessionID:       "conv-1",
		RootUserID:      1001,
		CreatorUserID:   1001,
		LastActorUserID: 1001,
		Title:           "历史测试",
		Status:          conversationStatusReady,
	})

	svc := NewService(db, nil, &config.Config{
		Agent: config.AgentConfig{Enabled: true, TimeoutMS: 1000},
	}, zap.NewNop(), fakeAgentClient{
		historyResp: &agentv1.GetSessionHistoryResponse{
			Turns: []*agentv1.ConversationTurn{
				{SequenceNo: 1, Role: "user", Content: "你好", CreatedAt: "2026-04-08T10:00:00Z", Domain: agentv1.Domain_DOMAIN_CHAT},
				{SequenceNo: 2, Role: "assistant", Content: "你好呀", CreatedAt: "2026-04-08T10:00:01Z", Domain: agentv1.Domain_DOMAIN_CHAT},
			},
			HasMore:              true,
			NextBeforeSequenceNo: 1,
		},
	})

	history, err := svc.GetHistory(context.Background(), 1001, "conv-1", nil, 20)
	if err != nil {
		t.Fatalf("GetHistory error: %v", err)
	}
	if history.ConversationID != "conv-1" {
		t.Fatalf("conversation id = %q, want conv-1", history.ConversationID)
	}
	if len(history.Turns) != 2 {
		t.Fatalf("turns len = %d, want 2", len(history.Turns))
	}
	if history.Turns[0].Domain != "chat" {
		t.Fatalf("turn domain = %q, want chat", history.Turns[0].Domain)
	}
	if !history.HasMore || history.NextBeforeSequenceNo != 1 {
		t.Fatalf("unexpected paging result: %+v", history)
	}

	if _, err := svc.GetHistory(context.Background(), 9999, "conv-1", nil, 20); err == nil {
		t.Fatalf("expected ownership error")
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema error: %v", err)
	}
	return db
}

func findConversation(t *testing.T, db *gorm.DB, sessionID string) *Conversation {
	t.Helper()

	var conversation Conversation
	err := db.WithContext(context.Background()).
		Where("session_id = ?", sessionID).
		Take(&conversation).Error
	if err != nil {
		t.Fatalf("find conversation: %v", err)
	}
	return &conversation
}

func saveConversation(t *testing.T, db *gorm.DB, conversation *Conversation) {
	t.Helper()

	if err := db.WithContext(context.Background()).Save(conversation).Error; err != nil {
		t.Fatalf("save conversation: %v", err)
	}
}
