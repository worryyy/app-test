package comment

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateCommentReq
	if !bindJSON(c, &req) {
		return
	}

	if _, err := h.svc.AddComment(c.Request.Context(), c.Param("topic_id"), middleware.GetUserID(c), req.Comment, req.ParentCmtID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.DeleteComment(c.Request.Context(), c.Param("topic_id"), c.Param("comment_id"), middleware.GetUserID(c), false); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) ListByTopic(c *gin.Context) {
	page, size := pageSize(c)
	rootID := firstNonEmpty(c.Query("root_id"), c.Query("rootId"))
	data, err := h.svc.ListByTopic(c.Request.Context(), c.Param("topic_id"), rootID, middleware.GetUserID(c), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) Mine(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListMine(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) TargetUserComments(c *gin.Context) {
	page, size := pageSize(c)
	targetUserID := firstNonEmpty(c.Query("target_user_id"), c.Query("targetUserId"))
	data, err := h.svc.ListTargetUserComments(c.Request.Context(), targetUserID, page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeCommentListResult(c, data, true)
}

func (h *Handler) Like(c *gin.Context) {
	if err := h.svc.LikeComment(c.Request.Context(), c.Param("comment_id"), middleware.GetUserID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Unlike(c *gin.Context) {
	if err := h.svc.UnlikeComment(c.Request.Context(), c.Param("comment_id"), middleware.GetUserID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
