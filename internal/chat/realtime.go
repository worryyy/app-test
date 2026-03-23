package chat

import (
	"context"
	"fmt"
	"strconv"
)

func (h *Handler) PushNotification(ctx context.Context, targetUserID string, payload interface{}) error {
	_ = ctx
	userID, err := strconv.ParseInt(targetUserID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}
	session, ok := h.sessions.Get(userID)
	if !ok || session == nil || session.Conn == nil {
		return nil
	}
	if err := session.Conn.WriteJSON(map[string]interface{}{
		"type": "notification",
		"data": payload,
	}); err != nil {
		return fmt.Errorf("write websocket notification: %w", err)
	}
	return nil
}
