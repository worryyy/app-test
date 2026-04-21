package agentchat

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const (
	wsHeartbeatInterval = 30 * time.Second
	wsSessionTimeout    = 60 * time.Second
	wsPingPayload       = "server_heartbeat"
)

type Session struct {
	RootUserID int64
	Conn       *websocket.Conn
	mu         sync.Mutex
}

func (s *Session) WriteJSON(v any) error {
	if s == nil || s.Conn == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteJSON(v)
}

func (s *Session) WriteMessage(messageType int, data []byte) error {
	if s == nil || s.Conn == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteMessage(messageType, data)
}

func (h *Handler) WS(c *gin.Context) {
	conn, err := upgradeWS(c)
	if err != nil {
		responses.Fail(c, bizerr.Biz("websocket upgrade failed"))
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	if err := configureWSConn(conn); err != nil {
		return
	}

	claims, err := h.handleWSAuth(c.Request.Context(), conn)
	if err != nil {
		_ = conn.WriteJSON(wsAuthFailedEvent(err))
		return
	}

	session := &Session{RootUserID: rootUserIDFromClaims(claims), Conn: conn}
	_ = h.serveWSConnection(c.Request.Context(), session, claims)
}

func upgradeWS(c *gin.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

func configureWSConn(conn *websocket.Conn) error {
	conn.SetReadLimit(1 << 20)
	if err := conn.SetReadDeadline(time.Now().Add(wsSessionTimeout)); err != nil {
		return err
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsSessionTimeout))
	})
	return nil
}

func wsAuthFailedEvent(err error) WSEvent {
	return WSEvent{
		Type:      "auth_failed",
		ErrorCode: "unauthorized",
		Message:   err.Error(),
	}
}

func rootUserIDFromClaims(claims *jwtutil.Claims) int64 {
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
	}
	return claims.UserID
}

func (h *Handler) serveWSConnection(ctx context.Context, session *Session, claims *jwtutil.Claims) error {
	baseCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stopPing := startWSPingLoop(session)
	defer close(stopPing)

	return h.readWSMessages(baseCtx, session, claims)
}
