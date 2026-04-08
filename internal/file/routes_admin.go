package file

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.POST("/file", handler.SetPublic)
	admin.GET("/file", handler.List)
}
