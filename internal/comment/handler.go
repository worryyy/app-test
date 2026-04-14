package comment

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	var req CreateCommentReq
	if !bindJSON(c, &req) {
		return
	}

	if _, err := h.svc.AddComment(c.Request.Context(), uri.TopicID, middleware.GetUserID(c), req.Comment, req.ParentCmtID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Delete(c *gin.Context) {
	var uri topicCommentURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteComment(c.Request.Context(), uri.TopicID, uri.CommentID, middleware.GetUserID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) ListByTopic(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	var query commentListQuery
	if !bindQuery(c, &query) {
		return
	}

	page, size := pageSize(c)
	data, err := h.svc.ListByTopic(c.Request.Context(), uri.TopicID, query.RootID, middleware.GetUserID(c), page, size)
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
	var query targetUserCommentsQuery
	if !bindQuery(c, &query) {
		return
	}

	page, size := pageSize(c)
	data, err := h.svc.ListTargetUserComments(c.Request.Context(), query.TargetUserID, page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeCommentListResult(c, data, true)
}

func (h *Handler) Like(c *gin.Context) {
	var uri commentURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.LikeComment(c.Request.Context(), uri.CommentID, middleware.GetUserID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Unlike(c *gin.Context) {
	var uri commentURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.UnlikeComment(c.Request.Context(), uri.CommentID, middleware.GetUserID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
