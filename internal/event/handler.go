package event

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Add(c *gin.Context) {
	var req Event
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.AddEvent(c.Request.Context(), &req, middleware.GetUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "添加成功", nil)
}
