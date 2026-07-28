package notification

import (
	"errors"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct {
	svc          *Service
	hub          *SessionHub
	jwtHelper    *jwtutil.Helper
	legacyPusher func(string, any) error
}

func NewHandler(svc *Service, jwtHelper *jwtutil.Helper) *Handler {
	return &Handler{svc: svc, hub: NewSessionHub(), jwtHelper: jwtHelper}
}

func (h *Handler) SetLegacyPusher(pusher func(string, any) error) {
	h.legacyPusher = pusher
}

func (h *Handler) List(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.List(c.Request.Context(), rootUserID(c), c.Query("category"), page, size)
	respond(c, data, err)
}

func (h *Handler) UnreadCounts(c *gin.Context) {
	data, err := h.svc.UnreadCounts(c.Request.Context(), rootUserID(c))
	respond(c, data, err)
}

func (h *Handler) MarkOneRead(c *gin.Context) {
	if err := h.svc.MarkOneRead(c.Request.Context(), rootUserID(c), c.Param("id")); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) MarkRead(c *gin.Context) {
	var req struct {
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		responses.Fail(c, bizerr.Param("请求参数错误"))
		return
	}
	count, err := h.svc.MarkRead(c.Request.Context(), rootUserID(c), req.Category)
	respond(c, gin.H{"updated": count}, err)
}

func (h *Handler) LegacyList(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListLegacy(c.Request.Context(), middleware.GetUserID(c), rootUserID(c), c.Query("type"), page, size)
	respond(c, data, err)
}

func (h *Handler) LegacyHaveUnread(c *gin.Context) {
	data, err := h.svc.HaveUnreadLegacy(c.Request.Context(), middleware.GetUserID(c), rootUserID(c), c.Param("type"))
	respond(c, data, err)
}

func (h *Handler) LegacyLatest(c *gin.Context) {
	data, err := h.svc.LatestLegacy(c.Request.Context(), middleware.GetUserID(c), rootUserID(c), c.Param("type"))
	respond(c, data, err)
}

func (h *Handler) Broadcast(event Broadcast) {
	h.hub.Broadcast(event.RootUserID, gin.H{"type": "notification", "data": event.Notification})
	if h.legacyPusher == nil {
		return
	}
	target := event.LegacyReceiverID
	if target == "" {
		target = strconv.FormatInt(event.RootUserID, 10)
	}
	_ = h.legacyPusher(target, map[string]any{
		"id": event.Legacy.ID, "receiverId": event.Legacy.ReceiverID,
		"senderId": event.Legacy.SenderID, "type": event.Legacy.Type,
		"content": event.Legacy.Content, "topicId": event.Legacy.TopicID,
		"commentId": event.Legacy.CommentID, "createdTime": event.Legacy.CreatedTime,
		"isRead": event.Legacy.IsRead,
	})
}

func rootUserID(c *gin.Context) int64 {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
	}
	return claims.UserID
}

func respond(c *gin.Context, data any, err error) {
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
