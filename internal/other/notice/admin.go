package notice

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) NoticeAdd(c *gin.Context) {
	var req Notice
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.CreateNotice(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) NoticeDelete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	if err := h.svc.DeleteNotice(c.Request.Context(), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "删除成功", nil)
}

func (h *AdminHandler) NoticeUpdate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	var req Notice
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateNotice(c.Request.Context(), id, &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "更新成功", nil)
}

func (h *AdminHandler) NoticeGet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	data, err := h.svc.GetNotice(c.Request.Context(), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) NoticeList(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListNotices(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
