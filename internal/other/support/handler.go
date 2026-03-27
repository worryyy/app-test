package support

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) SupportByKey(c *gin.Context) {
	data, err := h.svc.GetSupportByKey(c.Request.Context(), c.Param("key"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) SupportList(c *gin.Context) {
	data, err := h.svc.ListSupport(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
