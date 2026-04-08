package comment

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.DELETE("/comment/:topic_id/:comment_id", handler.Delete)
}
