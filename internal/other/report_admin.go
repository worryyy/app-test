package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *AdminHandler) ReportReview(c *gin.Context) {
	var req ReportReviewReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.ReviewReportComment(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), req.HandlerContent); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) ReportList(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListReportComments(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
