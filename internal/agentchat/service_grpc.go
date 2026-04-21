package agentchat

import (
	"strconv"
	"strings"

	agentv1 "github.com/Milchstrassse/Ecampus-go/internal/agentchat/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) mapCallError(err error) error {
	switch status.Code(err) {
	case codes.PermissionDenied:
		return ErrConversationAccessDenied
	case codes.NotFound:
		return ErrConversationNotFound
	case codes.InvalidArgument:
		return bizerr.Param(errMsgInvalidParam)
	case codes.DeadlineExceeded, codes.Unavailable:
		return ErrAgentUnavailable
	default:
		return ErrAgentUnavailable
	}
}

func (s *Service) wsErrorFromCallErr(err error) (string, string) {
	switch status.Code(err) {
	case codes.PermissionDenied:
		return "conversation_forbidden", ErrConversationAccessDenied.Message
	case codes.NotFound:
		return "conversation_not_found", ErrConversationNotFound.Message
	case codes.InvalidArgument:
		return "invalid_argument", errMsgInvalidParam
	case codes.DeadlineExceeded:
		return "deadline_exceeded", ErrAgentUnavailable.Message
	case codes.Unavailable:
		return "service_unavailable", ErrAgentUnavailable.Message
	default:
		return "service_unavailable", ErrAgentUnavailable.Message
	}
}

func (s *Service) streamErrorEvent(requestID, conversationID string, err error) WSEvent {
	code, message := s.wsErrorFromCallErr(err)
	return WSEvent{
		Type:           "agent_turn.error",
		RequestID:      requestID,
		ConversationID: conversationID,
		ErrorCode:      code,
		Message:        message,
	}
}

func buildMetadata(input TurnInput, requestID string) map[string]string {
	metadata := map[string]string{
		"source_app":         "ecampus-go",
		"root_user_id":       strconv.FormatInt(input.RootUserID, 10),
		"current_user_id":    strconv.FormatInt(input.CurrentUserID, 10),
		"account_type":       strings.TrimSpace(input.AccountType),
		"client_platform":    strings.TrimSpace(input.ClientPlatform),
		"school_id":          strings.TrimSpace(input.SchoolID),
		"school_name":        strings.TrimSpace(input.SchoolName),
		"locale":             strings.TrimSpace(input.Locale),
		"ecampus_request_id": requestID,
	}
	for key, value := range metadata {
		if strings.TrimSpace(value) == "" {
			delete(metadata, key)
		}
	}
	return metadata
}

func toTurnResult(resp *agentv1.HandleTurnResponse) TurnResult {
	if resp == nil {
		return TurnResult{ErrorCode: "unspecified"}
	}

	result := TurnResult{
		Domain:        enumValue(resp.GetDomain().String(), "DOMAIN_"),
		Intent:        enumValue(resp.GetIntent().String(), "INTENT_"),
		Mode:          enumValue(resp.GetMode().String(), "MODE_"),
		AnswerText:    resp.GetAnswerText(),
		TraceID:       resp.GetTraceId(),
		Degraded:      resp.GetDegraded(),
		DegradeReason: resp.GetDegradeReason(),
		ErrorCode:     enumValue(resp.GetErrorCode().String(), "ERROR_CODE_"),
	}
	if len(resp.GetReferences()) == 0 {
		return result
	}

	result.References = make([]TurnReference, 0, len(resp.GetReferences()))
	for _, ref := range resp.GetReferences() {
		result.References = append(result.References, TurnReference{
			Source: ref.GetSource(),
			Ref:    ref.GetRef(),
			Score:  ref.GetScore(),
		})
	}
	return result
}

func enumValue(value, prefix string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unspecified"
	}

	value = strings.ToLower(value)
	value = strings.TrimPrefix(value, strings.ToLower(prefix))
	if value == "" {
		return "unspecified"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
