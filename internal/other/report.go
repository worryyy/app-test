package other

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) ReportComment(c *gin.Context) {
	var req struct {
		CommentID string `json:"commentId" binding:"required"`
		TopicID   string `json:"topicId" binding:"required"`
		Reason    string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	_, err := h.svc.CreateReportComment(c.Request.Context(), &ReportComment{
		CommentID:  req.CommentID,
		TopicID:    req.TopicID,
		ReporterID: strconv.FormatInt(middleware.GetUserID(c), 10),
		Reason:     req.Reason,
		Status:     0,
	})
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
