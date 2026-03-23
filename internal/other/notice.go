package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) NoticeList(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListNotices(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
