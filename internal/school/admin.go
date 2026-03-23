package school

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) AddTerm(c *gin.Context) {
	var req Term
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	id, err := h.svc.AddTerm(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, id)
}

func (h *AdminHandler) DeleteTerm(c *gin.Context) {
	if err := h.svc.DeleteTerm(c.Request.Context(), c.Param("id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SetCurrentTerm(c *gin.Context) {
	var req CurTermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.SetCurrentTerm(c.Request.Context(), req.TermID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
