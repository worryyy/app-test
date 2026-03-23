package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) SupportByKey(c *gin.Context) {
	data, err := h.svc.GetSupportByKey(c.Request.Context(), c.Param("key"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) SupportList(c *gin.Context) {
	data, err := h.svc.ListSupport(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
