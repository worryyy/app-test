package theme

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

func (h *AdminHandler) Update(c *gin.Context) {
	var req ThemeUpdateReq
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.UpdateTheme(c.Request.Context(), c.Param("id"), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) List(c *gin.Context) {
	data, err := h.svc.ListThemes(c.Request.Context(), c.Query("name"))
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) UpdateSearch(c *gin.Context) {
	var req ThemeSearchReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.UpdateNeedSearch(c.Request.Context(), req.ThemeIDs, true); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) UpdateSuggest(c *gin.Context) {
	var req ThemeSuggestReq
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.UpdateSuggestByList(c.Request.Context(), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) AddCampusTheme(c *gin.Context) {
	var req CampusTheme
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.AddCampusTheme(c.Request.Context(), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) DeleteCampusTheme(c *gin.Context) {
	if err := h.svc.DeleteCampusTheme(c.Request.Context(), c.Param("themeId")); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "删除成功")
}
