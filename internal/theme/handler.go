package theme

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) InitCampusThemes(c *gin.Context) {
	var req []Theme
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.InitCampusThemes(c.Request.Context(), req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) GetCampusThemes(c *gin.Context) {
	data, err := h.svc.ListCampusThemes(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
