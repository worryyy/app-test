package chat

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
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
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) ConversationEnter(c *gin.Context) {
	var req ConversationEnterReq
	if !bindQuery(c, &req) {
		return
	}

	if err := h.svc.EnterConversation(c.Request.Context(), middleware.GetUserID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) ConversationUnreadCount(c *gin.Context) {
	var uri conversationIDURI
	if !bindURI(c, &uri) {
		return
	}

	data, err := h.svc.GetUnreadCount(c.Request.Context(), middleware.GetUserID(c), uri.ID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) ConversationQuery(c *gin.Context) {
	var query conversationQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.QueryConversation(c.Request.Context(), middleware.GetUserID(c), query.TargetUserID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) ProfileByConversationID(c *gin.Context) {
	var query conversationProfileQuery
	if !bindQuery(c, &query) {
		return
	}

	peerID, err := h.svc.GetPeerUserID(c.Request.Context(), query.ConversationID, middleware.GetUserID(c))
	if err != nil {
		responses.Fail(c, err)
		return
	}

	userID, err := strconv.ParseInt(peerID, 10, 64)
	if err != nil {
		responses.Fail(c, ErrConversationUserNotFound)
		return
	}
	if h.userSvc == nil {
		responses.Success.RespData(c, nil)
		return
	}

	profileUser, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	if profileUser == nil {
		responses.Fail(c, ErrConversationUserNotFound)
		return
	}

	responses.Success.RespData(c, &ConversationProfile{
		Avatar:    profileUser.Avatar,
		Nickname:  profileUser.Nickname,
		UserID:    strconv.FormatInt(profileUser.ID, 10),
		Gender:    profileUser.Gender,
		StuCla:    profileUser.StuCla,
		Signature: profileUser.Signature,
	})
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	var uri conversationIDURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteConversation(c.Request.Context(), middleware.GetUserID(c), uri.ID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "删除会话成功")
}

func (h *Handler) OfflineMessages(c *gin.Context) {
	var uri lastMessageURI
	if !bindURI(c, &uri) {
		return
	}

	lastMessageID, err := strconv.ParseInt(uri.LastMessageID, 10, 64)
	if err != nil {
		responses.ParamErr.RespMessage(c, "last_message_id格式错误")
		return
	}

	data, err := h.svc.GetOfflineMessages(c.Request.Context(), middleware.GetUserID(c), lastMessageID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) HistoryMessages(c *gin.Context) {
	var query historyMessagesQuery
	if !bindQuery(c, &query) {
		return
	}

	page, size := pageSize(c)
	oldestMessageID, err := parseOptionalPositiveInt64(query.OldestMessageID)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	data, err := h.svc.GetHistoryMessages(
		c.Request.Context(),
		middleware.GetUserID(c),
		query.ConversationID,
		oldestMessageID,
		page,
		size,
	)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) UnreadMessages(c *gin.Context) {
	data, err := h.svc.HasUnreadMessages(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) NotifyList(c *gin.Context) {
	var query notificationListQuery
	if !bindQuery(c, &query) {
		return
	}

	page, size := pageSize(c)
	data, err := h.svc.ListNotifications(c.Request.Context(), middleware.GetUserID(c), query.Type, page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeNotificationListResult(c, data)
}

func (h *Handler) NotifyHaveUnread(c *gin.Context) {
	var uri notificationTypeURI
	if !bindURI(c, &uri) {
		return
	}

	ok, err := h.svc.HaveUnreadNotification(c.Request.Context(), middleware.GetUserID(c), uri.Type)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, ok)
}

func (h *Handler) NotifyLatest(c *gin.Context) {
	var uri notificationTypeURI
	if !bindURI(c, &uri) {
		return
	}

	data, err := h.svc.LatestNotification(c.Request.Context(), middleware.GetUserID(c), uri.Type)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
