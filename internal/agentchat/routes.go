package agentchat

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	agent := api.Group("/agent")
	agent.GET("/conversation", handler.Conversations)
	agent.GET("/conversation/:id/history", handler.History)
	agent.DELETE("/conversation/:id", handler.DeleteConversation)
	agent.POST("/turn", handler.Turn)
}

func RegisterInfraRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/agent/ws", handler.WS)
}
