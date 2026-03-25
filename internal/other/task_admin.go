package other

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *AdminHandler) TaskAdd(c *gin.Context) {
	var req Task
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.CreateTask(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) TaskDelete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	if err := h.svc.DeleteTask(c.Request.Context(), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) TaskUpdate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	var req Task
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateTask(c.Request.Context(), id, &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) TaskGet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	data, err := h.svc.GetTask(c.Request.Context(), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) TaskList(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListTasks(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
