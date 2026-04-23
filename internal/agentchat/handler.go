package agentchat

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/ginutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct {
	svc       *Service
	jwtHelper *jwtutil.Helper
	redis     *redis.Client
}

func NewHandler(svc *Service, jwtHelper *jwtutil.Helper, redisClient *redis.Client) *Handler {
	return &Handler{
		svc:       svc,
		jwtHelper: jwtHelper,
		redis:     redisClient,
	}
}

func (h *Handler) Conversations(c *gin.Context) {
	page, size := ginutil.PageSize(c, maxConversationPageSize)
	data, err := h.svc.ListConversations(c.Request.Context(), currentRootUserID(c), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) History(c *gin.Context) {
	var uri conversationIDURI
	if !ginutil.BindURI(c, &uri, errMsgInvalidParam) {
		return
	}
	var query historyQuery
	if !ginutil.BindQuery(c, &query, errMsgInvalidParam) {
		return
	}
	beforeSequenceNo, err := ginutil.ParseOptionalPositiveInt64(query.BeforeSequenceNo, errMsgInvalidParam)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	data, err := h.svc.GetHistory(c.Request.Context(), currentRootUserID(c), uri.ID, beforeSequenceNo, query.Size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	var uri conversationIDURI
	if !ginutil.BindURI(c, &uri, errMsgInvalidParam) {
		return
	}
	if err := h.svc.DeleteConversation(c.Request.Context(), currentRootUserID(c), uri.ID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "删除 agent 会话成功")
}

func (h *Handler) Turn(c *gin.Context) {
	var req TurnRequest
	if !ginutil.BindJSON(c, &req, errMsgInvalidParam) {
		return
	}
	resp, err := h.svc.HandleTurn(c.Request.Context(), TurnInput{
		RequestID:      req.RequestID,
		ConversationID: req.ConversationID,
		Content:        req.Content,
		RootUserID:     currentRootUserID(c),
		CurrentUserID:  currentUserID(c),
		AccountType:    currentAccountType(c),
		ClientPlatform: req.ClientPlatform,
		SchoolID:       req.SchoolID,
		SchoolName:     req.SchoolName,
		Locale:         req.Locale,
	})
	if err != nil {
		responses.Fail(c, err)
		return
	}
	c.Set("request_id", resp.RequestID)
	responses.Success.RespData(c, resp)
}
