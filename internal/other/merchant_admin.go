package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

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
	result.Success(c, id)
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
	result.Success(c, data)
}
