package marketplace

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	group := api.Group("/marketplace")
	group.GET("/categories", handler.Categories)
	group.GET("/items", handler.SearchItems)
	group.GET("/items/mine", handler.MyItems)
	group.GET("/items/:id", handler.ItemDetail)
	group.POST("/items", handler.CreateItem)
	group.PUT("/items/:id", handler.UpdateItem)
	group.DELETE("/items/:id", handler.WithdrawItem)
	group.GET("/orders", handler.Orders)
	group.POST("/orders", handler.CreateOrder)
	group.POST("/orders/:id/pay", handler.Pay)
	group.PUT("/orders/:id/cancel", handler.CancelOrder)
	group.PUT("/orders/:id/delivered", handler.MarkDelivered)
	group.PUT("/orders/:id/received", handler.ConfirmReceived)
	group.POST("/orders/:id/refunds", handler.RequestRefund)
	group.POST("/orders/:id/disputes", handler.CreateDispute)
	group.POST("/payments/test/callback", handler.TestPaymentCallback)
}

func RegisterAdminRoutes(admin *gin.RouterGroup, handler *AdminHandler) {
	group := admin.Group("/marketplace")
	group.GET("/categories", handler.Categories)
	group.POST("/categories", handler.CreateCategory)
	group.PUT("/categories/:id", handler.UpdateCategory)
	group.GET("/orders", handler.Orders)
	group.GET("/refunds", handler.Refunds)
	group.PUT("/refunds/:id/decision", handler.DecideRefund)
	group.GET("/disputes", handler.Disputes)
	group.PUT("/disputes/:id/decision", handler.ResolveDispute)
	group.GET("/settlements", handler.Settlements)
	group.POST("/settlements/:id/retry", handler.RetrySettlement)
}
