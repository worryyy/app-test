package comment

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Delete(c *gin.Context) {
	var uri topicCommentURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteComment(c.Request.Context(), uri.TopicID, uri.CommentID, 0, true); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
