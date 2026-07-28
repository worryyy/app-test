package moderation

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	group := api.Group("/moderation")
	group.POST("/reports", handler.CreateReport)
	group.GET("/reports", handler.Reports)
	group.DELETE("/reports/:id", handler.WithdrawReport)
	group.GET("/punishments", handler.Punishments)
	group.POST("/punishments/:id/appeals", handler.CreateAppeal)
	group.GET("/appeals", handler.Appeals)
}

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	group := admin.Group("/moderation")
	group.GET("/reports", handler.Reports)
	group.PUT("/reports/:id/claim", handler.Claim)
	group.PUT("/reports/:id/decision", handler.Decide)
	group.PUT("/punishments/:id/revoke", handler.Revoke)
	group.GET("/appeals", handler.Appeals)
	group.PUT("/appeals/:id/decision", handler.DecideAppeal)
}
