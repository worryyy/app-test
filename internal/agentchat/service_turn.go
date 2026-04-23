package agentchat

import (
	"context"
	"io"
	"strconv"
	"strings"

	agentv1 "github.com/Milchstrassse/Ecampus-go/internal/agentchat/agentv1"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

type preparedTurn struct {
	conversation *Conversation
	requestID    string
	request      *agentv1.HandleTurnRequest
}

func (s *Service) HandleTurn(ctx context.Context, input TurnInput) (*TurnResponse, error) {
	prepared, release, err := s.prepareTurn(ctx, input)
	if err != nil {
		return nil, err
	}
	defer release()

	callCtx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	resp, err := s.client.HandleTurn(callCtx, prepared.request)
	if err != nil {
		return nil, s.mapCallError(err)
	}

	result := toTurnResult(resp)
	s.persistConversationResult(ctx, prepared.conversation, input, prepared.requestID, result)
	return &TurnResponse{
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		Result:         result,
	}, nil
}

func (s *Service) StreamTurn(ctx context.Context, input TurnInput, emit func(WSEvent) error) error {
	prepared, release, err := s.prepareTurn(ctx, input)
	if err != nil {
		return err
	}
	defer release()

	if err := emit(acceptedTurnEvent(prepared)); err != nil {
		return err
	}

	callCtx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	stream, err := s.client.StreamHandleTurn(callCtx, prepared.request)
	if err != nil {
		wsErr := s.streamErrorEvent(prepared.requestID, prepared.conversation.SessionID, err)
		return s.emitStreamFailure(ctx, prepared, input, emit, wsErr)
	}
	return s.consumeTurnStream(ctx, stream, prepared, input, emit)
}

func (s *Service) prepareTurn(ctx context.Context, input TurnInput) (preparedTurn, func(), error) {
	if strings.TrimSpace(input.Content) == "" {
		return preparedTurn{}, func() {}, ErrTurnContentRequired
	}

	conversation, err := s.resolveConversation(ctx, input)
	if err != nil {
		return preparedTurn{}, func() {}, err
	}

	requestID := normalizeTurnRequestID(input.RequestID)
	release, err := s.acquireTurnGuards(ctx, input.RootUserID, conversation.SessionID)
	if err != nil {
		return preparedTurn{}, func() {}, err
	}

	conversation.LastActorUserID = input.CurrentUserID
	conversation.LastRequestID = requestID
	conversation.Status = conversationStatusPending
	if conversation.Title == "" {
		conversation.Title = buildConversationTitle(input.Content)
	}
	if err := s.saveConversation(ctx, conversation); err != nil {
		release()
		return preparedTurn{}, func() {}, bizerr.InternalWrap("保存 agent 会话失败", err)
	}

	return preparedTurn{
		conversation: conversation,
		requestID:    requestID,
		request:      s.newHandleTurnRequest(input, conversation.SessionID, requestID),
	}, release, nil
}

func (s *Service) consumeTurnStream(
	ctx context.Context,
	stream agentv1.AgentService_StreamHandleTurnClient,
	prepared preparedTurn,
	input TurnInput,
	emit func(WSEvent) error,
) error {
	for {
		event, err := stream.Recv()
		if err != nil {
			return s.handleStreamReceiveError(ctx, prepared, input, emit, err)
		}

		done, err := s.handleStreamEvent(ctx, prepared, input, emit, event)
		if err != nil || done {
			return err
		}
	}
}

func (s *Service) handleStreamReceiveError(
	ctx context.Context,
	prepared preparedTurn,
	input TurnInput,
	emit func(WSEvent) error,
	err error,
) error {
	if err == io.EOF {
		return s.emitStreamFailure(ctx, prepared, input, emit, closedStreamEvent(prepared))
	}
	return s.emitStreamFailure(ctx, prepared, input, emit, s.streamErrorEvent(prepared.requestID, prepared.conversation.SessionID, err))
}

func (s *Service) handleStreamEvent(
	ctx context.Context,
	prepared preparedTurn,
	input TurnInput,
	emit func(WSEvent) error,
	event *agentv1.StreamHandleTurnEvent,
) (bool, error) {
	switch payload := event.Event.(type) {
	case *agentv1.StreamHandleTurnEvent_Status:
		return false, emit(statusTurnEvent(prepared, payload.Status.GetStage(), payload.Status.GetMessage()))
	case *agentv1.StreamHandleTurnEvent_Delta:
		return false, emit(deltaTurnEvent(prepared, payload.Delta.GetText()))
	case *agentv1.StreamHandleTurnEvent_FinalResponse:
		result := toTurnResult(payload.FinalResponse)
		s.persistConversationResult(ctx, prepared.conversation, input, prepared.requestID, result)
		return true, emit(finalTurnEvent(prepared, result))
	case *agentv1.StreamHandleTurnEvent_Error:
		wsErr := agentStreamErrorEvent(prepared, payload.Error.GetMessage())
		return true, s.emitStreamFailure(ctx, prepared, input, emit, wsErr)
	default:
		return true, s.emitStreamFailure(ctx, prepared, input, emit, protocolStreamErrorEvent(prepared))
	}
}

func (s *Service) emitStreamFailure(
	ctx context.Context,
	prepared preparedTurn,
	input TurnInput,
	emit func(WSEvent) error,
	wsErr WSEvent,
) error {
	s.persistConversationFailure(ctx, prepared.conversation, input, prepared.requestID, wsErr.Message)
	return emit(wsErr)
}

func (s *Service) persistConversationResult(ctx context.Context, conversation *Conversation, input TurnInput, requestID string, result TurnResult) {
	if conversation == nil {
		return
	}

	conversation.LastActorUserID = input.CurrentUserID
	conversation.LastRequestID = requestID
	conversation.LastTraceID = result.TraceID
	conversation.LastUserPreview = previewText(input.Content, 120)
	conversation.LastAssistantPreview = previewText(result.AnswerText, 120)
	conversation.Status = conversationStatusReady
	if result.ErrorCode != "unspecified" {
		conversation.Status = conversationStatusError
	}
	if conversation.Title == "" {
		conversation.Title = buildConversationTitle(input.Content)
	}
	if err := s.saveConversation(ctx, conversation); err != nil {
		s.logger.Warn("save agent conversation result failed", zap.String("conversation_id", conversation.SessionID), zap.Error(err))
	}
}

func (s *Service) persistConversationFailure(ctx context.Context, conversation *Conversation, input TurnInput, requestID, message string) {
	if conversation == nil {
		return
	}

	conversation.LastActorUserID = input.CurrentUserID
	conversation.LastRequestID = requestID
	conversation.LastUserPreview = previewText(input.Content, 120)
	conversation.LastAssistantPreview = previewText(message, 120)
	conversation.Status = conversationStatusError
	if conversation.Title == "" {
		conversation.Title = buildConversationTitle(input.Content)
	}
	if err := s.saveConversation(ctx, conversation); err != nil {
		s.logger.Warn("save agent conversation failure failed", zap.String("conversation_id", conversation.SessionID), zap.Error(err))
	}
}

func normalizeTurnRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID != "" {
		return requestID
	}
	return "agent-turn-" + snowflake.Generate().String()
}

func (s *Service) newHandleTurnRequest(input TurnInput, sessionID, requestID string) *agentv1.HandleTurnRequest {
	return &agentv1.HandleTurnRequest{
		RequestId:  requestID,
		UserId:     strconv.FormatInt(input.RootUserID, 10),
		SessionId:  sessionID,
		Utterance:  strings.TrimSpace(input.Content),
		DeadlineMs: s.timeout().Milliseconds(),
		Metadata:   buildMetadata(input, requestID),
	}
}

func acceptedTurnEvent(prepared preparedTurn) WSEvent {
	return WSEvent{
		Type:           "agent_turn.accepted",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
	}
}

func statusTurnEvent(prepared preparedTurn, stage, message string) WSEvent {
	return WSEvent{
		Type:           "agent_turn.status",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		Stage:          stage,
		Message:        message,
	}
}

func deltaTurnEvent(prepared preparedTurn, delta string) WSEvent {
	return WSEvent{
		Type:           "agent_turn.delta",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		Delta:          delta,
	}
}

func finalTurnEvent(prepared preparedTurn, result TurnResult) WSEvent {
	return WSEvent{
		Type:           "agent_turn.final",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		Result:         &result,
	}
}

func closedStreamEvent(prepared preparedTurn) WSEvent {
	return WSEvent{
		Type:           "agent_turn.error",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		ErrorCode:      "stream_closed",
		Message:        ErrAgentStreamProtocol.Message,
	}
}

func agentStreamErrorEvent(prepared preparedTurn, message string) WSEvent {
	return WSEvent{
		Type:           "agent_turn.error",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		ErrorCode:      "agent_error",
		Message:        firstNonEmpty(message, ErrAgentUnavailable.Message),
	}
}

func protocolStreamErrorEvent(prepared preparedTurn) WSEvent {
	return WSEvent{
		Type:           "agent_turn.error",
		RequestID:      prepared.requestID,
		ConversationID: prepared.conversation.SessionID,
		ErrorCode:      "stream_protocol_error",
		Message:        ErrAgentStreamProtocol.Message,
	}
}
