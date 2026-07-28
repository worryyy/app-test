package notification

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	group := api.Group("/notification")
	group.GET("", handler.List)
	group.GET("/unread-counts", handler.UnreadCounts)
	group.PUT("/:id/read", handler.MarkOneRead)
	group.PUT("/read", handler.MarkRead)

	legacy := api.Group("/notify")
	legacy.GET("", handler.LegacyList)
	legacy.GET("/:type/haveUnread", handler.LegacyHaveUnread)
	legacy.GET("/:type", handler.LegacyLatest)
}

func RegisterInfraRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/notification/ws", handler.WS)
}
