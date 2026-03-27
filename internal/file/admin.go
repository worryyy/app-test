package file

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
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
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	modified, err := h.svc.SetPublic(c.Request.Context(), ids, true)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if modified <= 0 {
		result.Fail(c, result.CodeFail, "失败")
		return
	}
	result.SuccessMsg(c, fmt.Sprintf("更改 %d 条记录", modified), nil)
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
