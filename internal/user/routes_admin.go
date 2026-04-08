package user

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.POST("/user", handler.AddUser)
	admin.POST("/user/add", handler.AddAdmin)
	admin.DELETE("/user/:id", handler.DeleteUser)
	admin.PUT("/user/:id", handler.EditUser)
	admin.GET("/user/:id", handler.GetUser)
	admin.GET("/user/list", handler.ListUsers)
	admin.POST("/user/clear", handler.ClearAuthentication)
	admin.POST("/user/course", handler.UserCourse)
}
