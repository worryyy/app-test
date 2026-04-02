package school

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"

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
	result.Data(c, data)
}

func (h *Handler) CurrentTerm(c *gin.Context) {
	data, err := h.svc.CurrentTerm(c.Request.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrCurrentTermNotConfigured):
			result.Fail(c, result.CodeNotExisted, "请联系管理员设置当前学期")
		case errors.Is(err, ErrCurrentTermInvalid):
			result.Fail(c, result.CodeNotExisted, "请联系管理员检查学期")
		default:
			result.HandleError(c, err)
		}
		return
	}
	result.Data(c, data)
}

func (h *Handler) CourseColor(c *gin.Context) {
	var req CourseColorReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SetCourseColor(c.Request.Context(), middleware.GetUserID(c), req.Colors); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
