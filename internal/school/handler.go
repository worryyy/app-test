package school

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

func (h *Handler) TermList(c *gin.Context) {
	data, err := h.svc.TermList(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) CurrentTerm(c *gin.Context) {
	data, err := h.svc.CurrentTerm(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) CourseColor(c *gin.Context) {
	var req CourseColorReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.SetCourseColor(c.Request.Context(), middleware.GetUserID(c), req.Colors); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
