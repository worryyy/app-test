package theme

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.PUT("/theme/:id", handler.Update)
	admin.GET("/theme", handler.List)
	admin.PUT("/theme/search", handler.UpdateSearch)
	admin.POST("/theme/campus", handler.AddCampusTheme)
	admin.DELETE("/theme/campus/:themeId", handler.DeleteCampusTheme)
}
