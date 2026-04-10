package sensitive

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) DeleteByWord(c *gin.Context) {
	var query wordQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.DeleteByWord(c.Request.Context(), query.Word); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Add(c *gin.Context) {
	var query wordQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.Add(c.Request.Context(), query.Word); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Page(c *gin.Context) {
	var query pageQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.FindByPage(c.Request.Context(), query.Page, query.Size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) SearchLike(c *gin.Context) {
	var query wordQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.FindByLike(c.Request.Context(), query.Word)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
