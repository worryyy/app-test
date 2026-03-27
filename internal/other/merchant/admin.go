package merchant

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) MerchantAdd(c *gin.Context) {
	var req MerchantThemeReq
	if !result.BindJSON(c, &req) {
		return
	}
	id, err := h.svc.AddMerchantTheme(c.Request.Context(), req.ThemeID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	doc := MerchantTheme{ThemeID: req.ThemeID}
	doc.ID, _ = primitive.ObjectIDFromHex(id)
	result.Data(c, doc)
}

func (h *AdminHandler) MerchantDelete(c *gin.Context) {
	if err := h.svc.DeleteMerchantTheme(c.Request.Context(), c.Param("id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) MerchantList(c *gin.Context) {
	data, err := h.svc.ListMerchantThemes(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) TaskAdd(c *gin.Context) {
	var req TaskNameReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.CreateTask(c.Request.Context(), req.Name); err != nil {
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
	var req TaskNameReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateTask(c.Request.Context(), id, req.Name); err != nil {
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
	result.Data(c, data)
}

func (h *AdminHandler) TaskList(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListTasks(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
