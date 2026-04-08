package theme

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

func (h *AdminHandler) Update(c *gin.Context) {
	var uri themeIDURI
	if !bindURI(c, &uri) {
		return
	}

	var req ThemeUpdateReq
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.UpdateTheme(c.Request.Context(), uri.ID, &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) List(c *gin.Context) {
	var query themeListQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.ListThemes(c.Request.Context(), query.Name)
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
	var uri campusThemeURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteCampusTheme(c.Request.Context(), uri.ThemeID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "删除成功")
}
