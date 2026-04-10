package school

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	admin.GET("/term", handler.CurrentTerm)
	admin.POST("/term", handler.AddTerm)
	admin.DELETE("/term/:id", handler.DeleteTerm)
	admin.POST("/term/cur", handler.SetCurrentTerm)
	admin.GET("/term/list", handler.ListTerms)
}
