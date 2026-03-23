package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *AdminHandler) ReportReview(c *gin.Context) {
	var req ReportReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.ReviewReportComment(c.Request.Context(), c.Param("id"), req.Status); err != nil {
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
	result.Success(c, data)
}
