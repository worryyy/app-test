package other

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) ReportComment(c *gin.Context) {
	var req struct {
		CommentID     string `json:"commentId" binding:"required"`
		ReportContent string `json:"reportContent" binding:"required"`
	}
	if !result.BindJSON(c, &req) {
		return
	}
	report, err := h.svc.CreateReportComment(c.Request.Context(), &ReportComment{
		CommentID:     req.CommentID,
		ReportContent: req.ReportContent,
		ReportUserID:  strconv.FormatInt(middleware.GetUserID(c), 10),
	})
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, report)
}
