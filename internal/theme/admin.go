package theme

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Update(c *gin.Context) {
	var req Theme
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateTheme(c.Request.Context(), c.Param("id"), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) List(c *gin.Context) {
	data, err := h.svc.ListThemes(c.Request.Context(), c.Query("name"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) UpdateSearch(c *gin.Context) {
	var req ThemeSearchReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateNeedSearch(c.Request.Context(), req.ThemeID, req.NeedSearch); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) UpdateSuggest(c *gin.Context) {
	var req ThemeSuggestReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateSuggestConfig(c.Request.Context(), req.ThemeID, Theme{
		NeedSuggest:       req.NeedSuggest,
		SuggestBasicScore: req.SuggestBasicScore,
		SuggestNumber:     req.SuggestNumber,
		SuggestSetName:    req.SuggestSetName,
		SuggestType:       req.SuggestType,
	}); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) AddCampusTheme(c *gin.Context) {
	var req CampusTheme
	if !result.BindJSON(c, &req) {
		return
	}
	data, err := h.svc.AddCampusTheme(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) DeleteCampusTheme(c *gin.Context) {
	deleted, err := h.svc.DeleteCampusTheme(c.Request.Context(), c.Param("themeId"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if !deleted {
		result.Fail(c, result.CodeFail, "")
		return
	}
	result.Success(c, "删除成功")
}
