package event

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Add(c *gin.Context) {
	var req Event
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.AddEvent(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
