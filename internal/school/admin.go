package school

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) AddTerm(c *gin.Context) {
	var req Term
	if !bindJSON(c, &req) {
		return
	}

	term, err := h.svc.AddTerm(c.Request.Context(), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, term)
}

func (h *AdminHandler) DeleteTerm(c *gin.Context) {
	var uri termIDURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteTerm(c.Request.Context(), uri.ID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) SetCurrentTerm(c *gin.Context) {
	var req CurTermReq
	if !bindJSON(c, &req) {
		return
	}

	curTerm, err := h.svc.SetCurrentTerm(c.Request.Context(), req.TermID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, curTerm)
}
