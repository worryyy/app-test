package agentchat

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

func startWSPingLoop(session *Session) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(wsHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := session.WriteMessage(websocket.PingMessage, []byte(wsPingPayload)); err != nil {
					return
				}
			}
		}
	}()
	return stop
}

func (h *Handler) readWSMessages(ctx context.Context, session *Session, claims *jwtutil.Claims) error {
	for {
		raw, err := readWSMessage(session.Conn)
		if err != nil {
			return err
		}
		if err := h.handleWSMessage(ctx, session, claims, raw); err != nil {
			return err
		}
	}
}

func readWSMessage(conn *websocket.Conn) ([]byte, error) {
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(wsSessionTimeout)); err != nil {
		return nil, err
	}
	return raw, nil
}

func (h *Handler) handleWSMessage(ctx context.Context, session *Session, claims *jwtutil.Claims, raw []byte) error {
	messageType, err := parseWSMessageType(raw)
	if err != nil {
		return session.WriteJSON(invalidWSEvent("invalid_message"))
	}

	switch messageType {
	case "auth":
		return session.WriteJSON(WSEvent{
			Type:       "auth_success",
			RootUserID: strconv.FormatInt(session.RootUserID, 10),
		})
	case "agent_turn.start":
		return h.startWSTurn(ctx, session, claims, raw)
	default:
		return session.WriteJSON(invalidWSEvent("unsupported_message"))
	}
}

func parseWSMessageType(raw []byte) (string, error) {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return "", err
	}
	return strings.TrimSpace(probe.Type), nil
}

func invalidWSEvent(code string) WSEvent {
	return WSEvent{
		Type:      "agent_turn.error",
		ErrorCode: code,
		Message:   ErrInvalidWSMessage.Message,
	}
}

func (h *Handler) startWSTurn(ctx context.Context, session *Session, claims *jwtutil.Claims, raw []byte) error {
	var req wsTurnStartRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return session.WriteJSON(invalidWSEvent("invalid_message"))
	}

	go h.handleWSTurn(ctx, session, claims, req)
	return nil
}

func (h *Handler) handleWSTurn(ctx context.Context, session *Session, claims *jwtutil.Claims, req wsTurnStartRequest) {
	if session == nil {
		return
	}

	err := h.svc.StreamTurn(ctx, turnInputFromWS(claims, req), func(event WSEvent) error {
		return session.WriteJSON(event)
	})
	if err == nil {
		return
	}
	_ = session.WriteJSON(rejectedWSEvent(req, err))
}

func turnInputFromWS(claims *jwtutil.Claims, req wsTurnStartRequest) TurnInput {
	input := TurnInput{
		RequestID:      req.RequestID,
		ConversationID: req.ConversationID,
		Content:        req.Content,
		ClientPlatform: req.ClientPlatform,
		SchoolID:       req.SchoolID,
		SchoolName:     req.SchoolName,
		Locale:         req.Locale,
	}
	if claims == nil {
		return input
	}

	input.RootUserID = rootUserIDFromClaims(claims)
	input.CurrentUserID = claims.UserID
	input.AccountType = claims.AccountType
	return input
}

func rejectedWSEvent(req wsTurnStartRequest, err error) WSEvent {
	return WSEvent{
		Type:           "agent_turn.error",
		RequestID:      strings.TrimSpace(req.RequestID),
		ConversationID: strings.TrimSpace(req.ConversationID),
		ErrorCode:      "request_rejected",
		Message:        err.Error(),
	}
}

func (h *Handler) handleWSAuth(ctx context.Context, conn *websocket.Conn) (*jwtutil.Claims, error) {
	if h.jwtHelper == nil {
		return nil, errors.New("jwt helper not configured")
	}

	_, raw, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var first wsAuthEnvelope
	if err := json.Unmarshal(raw, &first); err != nil {
		return nil, err
	}
	if first.Type != "auth" {
		return nil, ErrWSAuthFailed
	}
	if strings.TrimSpace(first.Token) == "" {
		return nil, ErrWSAuthTokenRequired
	}

	claims, err := h.jwtHelper.ParseAndVerify(ctx, first.Token, h.redis)
	if err != nil {
		return nil, err
	}
	if err := conn.WriteJSON(WSEvent{
		Type:       "auth_success",
		RootUserID: strconv.FormatInt(rootUserIDFromClaims(claims), 10),
	}); err != nil {
		return nil, err
	}
	return claims, nil
}
