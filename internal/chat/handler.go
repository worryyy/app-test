package chat

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type Handler struct {
	svc       *Service
	userSvc   *user.Service
	jwtHelper *jwtutil.Helper
	redis     *redis.Client
	sessions  *SessionManager
}

func NewHandler(svc *Service, userSvc *user.Service, jwtHelper *jwtutil.Helper, redisClient *redis.Client) *Handler {
	return &Handler{
		svc:       svc,
		userSvc:   userSvc,
		jwtHelper: jwtHelper,
		redis:     redisClient,
		sessions:  NewSessionManager(),
	}
}

func (h *Handler) Conversations(c *gin.Context) {
	data, err := h.svc.ListConversations(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) ConversationEnter(c *gin.Context) {
	var req ConversationEnterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.EnterConversation(c.Request.Context(), middleware.GetUserID(c), req.ConversationID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) ConversationUnreadCount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	count, err := h.svc.GetUnreadCount(c.Request.Context(), middleware.GetUserID(c), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, count)
}

func (h *Handler) ConversationQuery(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.QueryConversation(c.Request.Context(), middleware.GetUserID(c), targetUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) ProfileByConversationID(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Query("conversationId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	peerID, err := h.svc.GetPeerUserID(c.Request.Context(), conversationID, middleware.GetUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	var data interface{}
	if h.userSvc != nil && peerID > 0 {
		data, err = h.userSvc.GetByID(c.Request.Context(), peerID)
		if err != nil {
			result.HandleError(c, err)
			return
		}
	}
	result.Success(c, data)
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.DeleteConversation(c.Request.Context(), middleware.GetUserID(c), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) OfflineMessages(c *gin.Context) {
	lastMessageID, err := strconv.ParseInt(c.Param("last_message_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GetOfflineMessages(c.Request.Context(), middleware.GetUserID(c), lastMessageID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) HistoryMessages(c *gin.Context) {
	page, size := pageSize(c)
	conversationID, err := strconv.ParseInt(c.Query("conversationId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GetHistoryMessages(c.Request.Context(), conversationID, page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) UnreadMessages(c *gin.Context) {
	data, err := h.svc.GetUnreadMessages(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) NotifyList(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListNotifications(c.Request.Context(), middleware.GetUserID(c), c.Query("type"), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) NotifyHaveUnread(c *gin.Context) {
	ok, err := h.svc.HaveUnreadNotification(c.Request.Context(), middleware.GetUserID(c), c.Param("type"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, ok)
}

func (h *Handler) NotifyLatest(c *gin.Context) {
	data, err := h.svc.LatestNotification(c.Request.Context(), middleware.GetUserID(c), c.Param("type"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func pageSize(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	return page, size
}
