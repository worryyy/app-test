package support

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

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
	req.ID, _ = primitive.ObjectIDFromHex(id)
	result.Data(c, req)
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
	saved, err := h.svc.GetSupportByKey(c.Request.Context(), req.Key)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, saved)
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
	result.Data(c, data)
}
