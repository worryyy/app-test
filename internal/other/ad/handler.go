package ad

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) AdListByLevel(c *gin.Context) {
	size, _ := strconv.Atoi(c.DefaultQuery("size", "0"))
	data, err := h.svc.ListAdByLevel(c.Request.Context(), size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
