package topic

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) List(c *gin.Context) {
	page, size := requestPageSize(c)

	data, err := h.svc.ListAdmin(c.Request.Context(), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) Create(c *gin.Context) {
	var req CreateTopicReq
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.CreateAdmin(c.Request.Context(), middleware.GetAdminClaims(c), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) Update(c *gin.Context) {
	var uri resourceIDURI
	if !bindURI(c, &uri) {
		return
	}

	var req UpdateTopicReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.UpdateAdmin(c.Request.Context(), uri.ID, &req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Delete(c *gin.Context) {
	var uri resourceIDURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteAdmin(c.Request.Context(), uri.ID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
