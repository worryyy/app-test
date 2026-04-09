package sensitive

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	group := admin.Group("/sensitive")
	group.GET("/getAllList", handler.GetAllList)
	group.GET("/getByWord", handler.GetByWord)
	group.GET("/getByWord/", handler.GetByWord)
	group.DELETE("/deleteByWord", handler.DeleteByWord)
	group.DELETE("/batchDelete", handler.BatchDelete)
	group.POST("/add", handler.Add)
	group.POST("/batchAdd", handler.BatchAdd)
	group.GET("/page", handler.Page)
	group.GET("/search_like", handler.SearchLike)
	group.PUT("/update", handler.Update)
}
