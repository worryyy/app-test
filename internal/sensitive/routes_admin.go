package sensitive

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	group := admin.Group("/sensitive")
	group.DELETE("/deleteByWord", handler.DeleteByWord)
	group.POST("/add", handler.Add)
	group.GET("/page", handler.Page)
	group.GET("/search_like", handler.SearchLike)
}
