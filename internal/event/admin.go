package event

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.DeleteEvent(c.Request.Context(), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	var req Event
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.UpdateEvent(c.Request.Context(), id, &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GetEvent(c.Request.Context(), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	data, err := h.svc.ListEvents(c.Request.Context(), page, size, c.Query("eventType"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
