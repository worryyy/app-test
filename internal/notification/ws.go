package notification

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	wsHeartbeatInterval = 30 * time.Second
	wsSessionTimeout    = 60 * time.Second
)

type wsSession struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *wsSession) writeJSON(value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteJSON(value)
}

type SessionHub struct {
	mu       sync.RWMutex
	sessions map[int64]map[*wsSession]struct{}
}

func NewSessionHub() *SessionHub {
	return &SessionHub{sessions: make(map[int64]map[*wsSession]struct{})}
}

func (h *SessionHub) Add(rootUserID int64, session *wsSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions[rootUserID] == nil {
		h.sessions[rootUserID] = make(map[*wsSession]struct{})
	}
	h.sessions[rootUserID][session] = struct{}{}
}

func (h *SessionHub) Remove(rootUserID int64, session *wsSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions[rootUserID], session)
	if len(h.sessions[rootUserID]) == 0 {
		delete(h.sessions, rootUserID)
	}
}

func (h *SessionHub) Broadcast(rootUserID int64, payload any) {
	h.mu.RLock()
	sessions := make([]*wsSession, 0, len(h.sessions[rootUserID]))
	for session := range h.sessions[rootUserID] {
		sessions = append(sessions, session)
	}
	h.mu.RUnlock()
	for _, session := range sessions {
		_ = session.writeJSON(payload)
	}
}

type wsAuthEnvelope struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (h *Handler) WS(c *gin.Context) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(wsSessionTimeout))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(wsSessionTimeout)) })

	rootUserID, err := h.authenticateWS(c, conn)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "auth_failed", "msg": err.Error()})
		return
	}
	session := &wsSession{conn: conn}
	h.hub.Add(rootUserID, session)
	defer h.hub.Remove(rootUserID, session)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(wsHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				session.mu.Lock()
				err := conn.WriteMessage(websocket.PingMessage, []byte("server_heartbeat"))
				session.mu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()
	defer close(done)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(wsSessionTimeout))
	}
}

func (h *Handler) authenticateWS(c *gin.Context, conn *websocket.Conn) (int64, error) {
	if h.svc == nil || h.svc.redis == nil || h.jwtHelper == nil {
		return 0, errors.New("notification websocket not configured")
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	var auth wsAuthEnvelope
	if err := json.Unmarshal(raw, &auth); err != nil {
		return 0, err
	}
	if auth.Type != "auth" || auth.Token == "" {
		return 0, errors.New("missing auth message")
	}
	claims, err := h.jwtHelper.ParseAndVerify(c.Request.Context(), auth.Token, h.svc.redis)
	if err != nil {
		return 0, err
	}
	rootUserID := claims.RootUserID
	if rootUserID <= 0 {
		rootUserID = claims.UserID
	}
	if err := conn.WriteJSON(gin.H{"type": "auth_success", "rootUserId": rootUserID}); err != nil {
		return 0, err
	}
	return rootUserID, nil
}
