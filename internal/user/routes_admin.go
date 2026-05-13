package user

import "github.com/gin-gonic/gin"

func RegisterAdminPublicRoutes(engine *gin.Engine, handler *AdminHandler) {
	group := engine.Group("/admin/user")
	group.POST("/login", handler.Login)
	group.POST("/refresh", handler.RefreshToken)
}

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.POST("/user/logout", handler.Logout)
	admin.POST("/user/mock_token", handler.UserToken)
	admin.POST("/user", handler.AddUser)
	admin.PUT("/user/:id", handler.EditUser)
	admin.GET("/user/list", handler.ListUsers)
	admin.PUT("/user/pre_authentication", handler.PreAuth)
	admin.PUT("/user/pre_authentication/batch", handler.PreAuthBatch)
	admin.POST("/user/clear", handler.ClearAuthentication)
}
