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

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Session struct {
	UserID int64
	Conn   *websocket.Conn
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

func (m *SessionManager) Set(userID int64, s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[userID] = s
}

func (m *SessionManager) Get(userID int64) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.sessions[userID]
	return v, ok
}

func (m *SessionManager) Remove(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, userID)
}

type wsEnvelope struct {
	Type           string `json:"type"`
	Token          string `json:"token"`
	ConversationID int64  `json:"conversationId"`
	ReceiverID     int64  `json:"receiverId"`
	Content        string `json:"content"`
}

func (h *Handler) WS(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		result.Fail(c, result.CodeFail, "websocket upgrade failed")
		return
	}

	conn.SetReadLimit(1 << 20)
	if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		if closeErr := conn.Close(); closeErr != nil && h.svc != nil && h.svc.logger != nil {
			h.svc.logger.Warn("close ws conn after set deadline failed")
		}
		return
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			return err
		}
		return nil
	})

	userID, err := h.handleWSAuth(c, conn)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "auth_failed", "msg": err.Error()})
		_ = conn.Close()
		return
	}

	s := &Session{UserID: userID, Conn: conn}
	h.sessions.Set(userID, s)
	defer func() {
		h.sessions.Remove(userID)
		_ = conn.Close()
	}()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if writeErr := conn.WriteMessage(websocket.PingMessage, []byte("ping")); writeErr != nil {
				return
			}
			<-pingTicker.C
		}
	}()

	for {
		_, msg, readErr := conn.ReadMessage()
		if readErr != nil {
			return
		}
		var env wsEnvelope
		if unmarshalErr := json.Unmarshal(msg, &env); unmarshalErr != nil {
			_ = conn.WriteJSON(gin.H{"type": "error", "msg": "invalid message"})
			continue
		}
		if env.Type == "auth" {
			_ = conn.WriteJSON(gin.H{"type": "auth_success", "userId": strconv.FormatInt(userID, 10)})
			continue
		}
		if env.Type != "message" {
			_ = conn.WriteJSON(gin.H{"type": "error", "msg": "unsupported message type"})
			continue
		}

		data, handleErr := h.svc.HandleMessage(c.Request.Context(), env.ConversationID, userID, env.ReceiverID, env.Content)
		if handleErr != nil {
			_ = conn.WriteJSON(gin.H{"type": "error", "msg": handleErr.Error()})
			continue
		}
		_ = conn.WriteJSON(gin.H{"type": "message_ack", "data": data})

		if peer, ok := h.sessions.Get(env.ReceiverID); ok {
			_ = peer.Conn.WriteJSON(gin.H{"type": "message", "data": data})
		}
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

	var first wsEnvelope
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
	_ = conn.WriteJSON(gin.H{"type": "auth_success", "userId": strconv.FormatInt(claims.UserID, 10)})
	return claims.UserID, nil
}
