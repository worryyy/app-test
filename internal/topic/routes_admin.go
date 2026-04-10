package topic

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	group := admin.Group("/topic")
	group.GET("", handler.List)
	group.POST("", handler.Create)
	group.PATCH("/:topic_id", handler.Update)
	group.DELETE("/:topic_id", handler.Delete)
}
