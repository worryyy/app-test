package topic

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("topic_id"), 0, true); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) RefreshSuggest(c *gin.Context) {
	version, err := h.svc.RefreshSuggest(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, version)
}
