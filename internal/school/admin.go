package school

import "github.com/gin-gonic/gin"

import 	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"


type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) AddTerm(c *gin.Context) {
	var req Term
	if !result.BindJSON(c, &req) {
		return
	}
	term, err := h.svc.AddTerm(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, term)
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
	if !result.BindJSON(c, &req) {
		return
	}
	curTerm, err := h.svc.SetCurrentTerm(c.Request.Context(), req.TermID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, curTerm)
}
