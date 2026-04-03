package file

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) SetPublic(c *gin.Context) {
	ids := c.QueryArray("img_list")
	if len(ids) == 0 {
		responses.ParamErr.RespMessage(c, "img_list不能为空")
		return
	}

	modified, err := h.svc.SetPublic(c.Request.Context(), ids, true)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, fmt.Sprintf("更改 %d 条记录", modified))
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
