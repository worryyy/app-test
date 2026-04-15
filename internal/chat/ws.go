package chat

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const (
	wsHeartbeatInterval = 30 * time.Second
	wsSessionTimeout    = 60 * time.Second
	wsPingPayload       = "server_heartbeat"
)

type Session struct {
	UserID int64
	Conn   *websocket.Conn
	mu     sync.Mutex
}

func (s *Session) WriteJSON(v interface{}) error {
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

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*Session),
	}
}

func (m *SessionManager) Set(userID int64, s *Session) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := m.sessions[userID]
	m.sessions[userID] = s
	return previous
}

func (m *SessionManager) Get(userID int64) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[userID]
	return session, ok
}

func (m *SessionManager) RemoveIfSame(userID int64, s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.sessions[userID]; ok && current == s {
		delete(m.sessions, userID)
	}
}

func newWSAuthSuccess(userID int64) gin.H {
	value := strconv.FormatInt(userID, 10)
	return gin.H{
		"type":    "auth_success",
		"userId":  value,
		"user_id": value,
	}
}

type wsAuthEnvelope struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (h *Handler) WS(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		responses.Fail(c, bizerr.Biz("websocket upgrade failed"))
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	conn.SetReadLimit(1 << 20)
	if err := conn.SetReadDeadline(time.Now().Add(wsSessionTimeout)); err != nil {
		return
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsSessionTimeout))
	})

	userID, err := h.handleWSAuth(c, conn)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "auth_failed", "msg": err.Error()})
		return
	}

	session := &Session{UserID: userID, Conn: conn}
	if previous := h.sessions.Set(userID, session); previous != nil && previous != session && previous.Conn != nil {
		_ = previous.Conn.Close()
	}
	defer h.sessions.RemoveIfSame(userID, session)

	stopPing := make(chan struct{})
	go func() {
		ticker := time.NewTicker(wsHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-ticker.C:
				if err := session.WriteMessage(websocket.PingMessage, []byte(wsPingPayload)); err != nil {
					return
				}
			}
		}
	}()
	defer close(stopPing)

	for {
		_, raw, readErr := conn.ReadMessage()
		if readErr != nil {
			return
		}
		if err := conn.SetReadDeadline(time.Now().Add(wsSessionTimeout)); err != nil {
			return
		}

		var messageTypeProbe map[string]interface{}
		if err := json.Unmarshal(raw, &messageTypeProbe); err != nil {
			_ = session.WriteJSON(gin.H{"type": "error", "msg": "invalid message"})
			continue
		}
		if stringField(messageTypeProbe, "type") == "auth" {
			_ = session.WriteJSON(newWSAuthSuccess(userID))
			continue
		}

		message, err := h.svc.HandleWSMessage(c.Request.Context(), userID, raw)
		if err != nil {
			_ = session.WriteJSON(gin.H{"type": "error", "msg": err.Error()})
			continue
		}

		receiverID, err := strconv.ParseInt(message.ReceiverID, 10, 64)
		if err != nil {
			continue
		}
		peer, ok := h.sessions.Get(receiverID)
		if !ok {
			continue
		}
		_ = peer.WriteJSON(message)
	}
}

func (h *Handler) handleWSAuth(c *gin.Context, conn *websocket.Conn) (int64, error) {
	if h.jwtHelper == nil {
		return 0, errors.New("jwt helper not configured")
	}

	_, raw, err := conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	var first wsAuthEnvelope
	if err := json.Unmarshal(raw, &first); err != nil {
		return 0, err
	}
	if first.Type != "auth" || first.Token == "" {
		return 0, errors.New("missing auth message")
	}

	claims, err := h.jwtHelper.ParseAndVerify(c.Request.Context(), first.Token, h.redis)
	if err != nil {
		return 0, err
	}
	if err := conn.WriteJSON(newWSAuthSuccess(claims.UserID)); err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
