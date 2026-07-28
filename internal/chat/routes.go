package chat

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	registerConversationRoutes(api, handler)
	registerMessageRoutes(api, handler)
}

func RegisterInfraRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/chat", handler.WS)
}

func registerConversationRoutes(api *gin.RouterGroup, handler *Handler) {
	conversation := api.Group("/conversation")
	conversation.GET("", handler.Conversations)
	conversation.PUT("/conversation_enter", handler.ConversationEnter)
	conversation.GET("/:id/unread_count", handler.ConversationUnreadCount)
	conversation.GET("/conversation_query", handler.ConversationQuery)
	conversation.GET("/profile_by_conversation_id", handler.ProfileByConversationID)
	conversation.DELETE("/:id", handler.DeleteConversation)
}

func registerMessageRoutes(api *gin.RouterGroup, handler *Handler) {
	message := api.Group("/message")
	message.GET("/:last_message_id", handler.OfflineMessages)
	message.GET("/history_messages", handler.HistoryMessages)
	message.GET("/unread_messages", handler.UnreadMessages)
}
