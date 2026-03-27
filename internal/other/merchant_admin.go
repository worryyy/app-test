package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
