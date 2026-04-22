package agentchat

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const (
	wsHeartbeatInterval = 30 * time.Second
	wsSessionTimeout    = 60 * time.Second
	wsWriteTimeout      = 10 * time.Second
	wsPingPayload       = "server_heartbeat"
)

type Session struct {
	RootUserID int64
	Conn       *websocket.Conn
	mu         sync.Mutex
	wg         sync.WaitGroup
	inflight   atomic.Int32
}

func (s *Session) WriteJSON(v any) error {
	if s == nil || s.Conn == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.Conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return s.Conn.WriteJSON(v)
}

func (s *Session) WriteMessage(messageType int, data []byte) error {
	if s == nil || s.Conn == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.Conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return s.Conn.WriteMessage(messageType, data)
}

func (h *Handler) WS(c *gin.Context) {
	conn, err := upgradeWS(c)
	if err != nil {
		responses.Fail(c, bizerr.Biz("websocket upgrade failed"))
		return
	}

	if err := configureWSConn(conn); err != nil {
		_ = conn.Close()
		return
	}

	claims, err := h.handleWSAuth(c.Request.Context(), conn)
	if err != nil {
		h.wsLogger().Warn("agent ws auth failed", zap.Error(err))
		_ = conn.WriteJSON(wsAuthFailedEvent())
		_ = conn.Close()
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

func wsAuthFailedEvent() WSEvent {
	return WSEvent{
		Type:      "auth_failed",
		ErrorCode: "unauthorized",
		Message:   "鉴权失败",
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
	stopPing := startWSPingLoop(session)
	err := h.readWSMessages(baseCtx, session, claims)
	cancel()
	close(stopPing)
	session.waitAsync()
	if session != nil && session.Conn != nil {
		_ = session.Conn.Close()
	}
	return err
}

func (s *Session) tryStartTurn(limit int) bool {
	if s == nil {
		return false
	}
	if limit <= 0 {
		limit = 1
	}

	maxInflight := int32(limit)
	for {
		current := s.inflight.Load()
		if current >= maxInflight {
			return false
		}
		if s.inflight.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (s *Session) finishTurn() {
	if s == nil {
		return
	}
	s.inflight.Add(-1)
}

func (s *Session) startAsync(fn func()) {
	if s == nil || fn == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func (s *Session) waitAsync() {
	if s == nil {
		return
	}
	s.wg.Wait()
}

func (h *Handler) wsLogger() *zap.Logger {
	if h != nil && h.svc != nil && h.svc.logger != nil {
		return h.svc.logger
	}
	return zap.NewNop()
}
