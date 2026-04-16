package theme

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	theme := api.Group("/theme")
	theme.POST("/campus/init", handler.InitCampusThemes)
	theme.GET("/campus", handler.GetCampusThemes)
}
