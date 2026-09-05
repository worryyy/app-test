package theme

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) InitCampusThemes(c *gin.Context) {
	data, err := h.svc.InitCampusThemes(c.Request.Context())
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) GetCampusThemes(c *gin.Context) {
	data, err := h.svc.ListCampusThemes(c.Request.Context())
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
// bench: single-service impact probe 1788631920
