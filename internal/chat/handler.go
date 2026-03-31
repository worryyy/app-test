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
	req := ConversationEnterReq{
		ConversationID: c.Query("conversation_id"),
		LastMessageID:  c.Query("last_message_id"),
	}
	if err := h.svc.EnterConversation(c.Request.Context(), middleware.GetUserID(c), req.ConversationID, req.LastMessageID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) ConversationUnreadCount(c *gin.Context) {
	data, err := h.svc.GetUnreadCount(c.Request.Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) ConversationQuery(c *gin.Context) {
	data, err := h.svc.QueryConversation(c.Request.Context(), middleware.GetUserID(c), c.Query("target_user_id"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) ProfileByConversationID(c *gin.Context) {
	peerID, err := h.svc.GetPeerUserID(c.Request.Context(), c.Query("conversation_id"), middleware.GetUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	userID, err := strconv.ParseInt(peerID, 10, 64)
	if err != nil {
		result.HandleError(c, result.NewBizError(result.CodeNotExisted, "会话中用户信息未找到"))
		return
	}
	if h.userSvc == nil {
		result.Data(c, nil)
		return
	}

	profileUser, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if profileUser == nil {
		result.HandleError(c, result.NewBizError(result.CodeNotExisted, "会话中用户信息未找到"))
		return
	}
	result.Data(c, &ConversationProfile{
		Avatar:    profileUser.Avatar,
		Nickname:  profileUser.Nickname,
		UserID:    strconv.FormatInt(profileUser.ID, 10),
		Gender:    profileUser.Gender,
		StuCla:    profileUser.StuCla,
		Signature: profileUser.Signature,
	})
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	if err := h.svc.DeleteConversation(c.Request.Context(), middleware.GetUserID(c), c.Param("id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "删除会话成功", nil)
}

func (h *Handler) OfflineMessages(c *gin.Context) {
	lastMessageID, err := strconv.ParseInt(c.Param("last_message_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
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
	var oldestMessageID *int64
	if rawOldest := c.Query("oldest_message_id"); rawOldest != "" {
		parsed, err := strconv.ParseInt(rawOldest, 10, 64)
		if err != nil {
			result.Fail(c, result.CodeParamError, result.ErrParam.Error())
			return
		}
		oldestMessageID = &parsed
	}

	data, err := h.svc.GetHistoryMessages(
		c.Request.Context(),
		middleware.GetUserID(c),
		c.Query("conversation_id"),
		oldestMessageID,
		page,
		size,
	)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) UnreadMessages(c *gin.Context) {
	data, err := h.svc.HasUnreadMessages(c.Request.Context(), middleware.GetUserID(c))
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
	if data == nil || len(data.Data) == 0 {
		result.Data(c, []Notification{})
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
	result.Data(c, data)
}

func pageSize(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	} else if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}
