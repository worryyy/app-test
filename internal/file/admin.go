package file

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) SetPublic(c *gin.Context) {
	var query fileSetPublicQuery
	if !bindQuery(c, &query) {
		return
	}

	modified, err := h.svc.SetPublic(c.Request.Context(), query.ImgList, true)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, fmt.Sprintf("修改 %d 条记录", modified))
}

func (h *AdminHandler) List(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListAll(c.Request.Context(), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
