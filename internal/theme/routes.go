package theme

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	api.POST("/campus/init", handler.InitCampusThemes)
	api.GET("/campus", handler.GetCampusThemes)
}
