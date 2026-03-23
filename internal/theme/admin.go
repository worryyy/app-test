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
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
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
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
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
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
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
	var req Theme
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	id, err := h.svc.AddTheme(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, id)
}

func (h *AdminHandler) DeleteCampusTheme(c *gin.Context) {
	if err := h.svc.DeleteTheme(c.Request.Context(), c.Param("themeId")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
