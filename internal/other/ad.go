package other

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) AdListByLevel(c *gin.Context) {
	level, _ := strconv.Atoi(c.DefaultQuery("level", "0"))
	data, err := h.svc.ListAdByLevel(c.Request.Context(), level)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
