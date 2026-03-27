package file

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) SetPublic(c *gin.Context) {
	var req FilePublicReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SetPublic(c.Request.Context(), req.MD5List, true); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) List(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListAll(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
