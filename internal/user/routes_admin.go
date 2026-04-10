package user

import "github.com/gin-gonic/gin"

func RegisterAdminAuthOnlyRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.PUT("/user/pre_authentication", handler.PreAuth)
}

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.POST("/user", handler.AddUser)
	admin.PUT("/user/:id", handler.EditUser)
	admin.GET("/user/list", handler.ListUsers)
	admin.POST("/user/clear", handler.ClearAuthentication)
}
