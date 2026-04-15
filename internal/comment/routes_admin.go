package comment

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.DELETE("/topic/:topic_id/comment/:comment_id", handler.Delete)
	admin.GET("/topic/:topic_id/comment", handler.List)
}
