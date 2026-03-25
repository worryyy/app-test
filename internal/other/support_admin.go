package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *AdminHandler) SupportAdd(c *gin.Context) {
	var req FrontendSupport
	if !result.BindJSON(c, &req) {
		return
	}
	id, err := h.svc.AddSupport(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, id)
}

func (h *AdminHandler) SupportUpdate(c *gin.Context) {
	var req FrontendSupport
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateSupport(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SupportDelete(c *gin.Context) {
	if err := h.svc.DeleteSupport(c.Request.Context(), c.Param("id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SupportList(c *gin.Context) {
	data, err := h.svc.ListSupport(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
