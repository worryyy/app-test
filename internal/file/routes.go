package file

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/file/:md5", handler.Download)
	engine.GET("/file", handler.ListPublic)
}

func RegisterProtectedRoutes(fileAuth *gin.RouterGroup, handler *Handler) {
	fileAuth.POST("/upload", handler.Upload)
	fileAuth.DELETE("/del/:md5", handler.Delete)
}
