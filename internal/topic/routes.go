package topic

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	registerTopicRoutes(api, handler)
	registerTopicLikeRoutes(api, handler)
	registerTopicCollectionRoutes(api, handler)
}

func registerTopicRoutes(api *gin.RouterGroup, handler *Handler) {
	topic := api.Group("/topic")
	topic.POST("", handler.Create)
	topic.DELETE("/:id", handler.Delete)
	topic.GET("/:topic_id", handler.GetByID)
	topic.PUT("/:topic_id", handler.Update)
	topic.GET("/search", handler.Search)
	topic.GET("", handler.Mine)
	topic.GET("/target_user_topics", handler.TargetUserTopics)
}

func registerTopicLikeRoutes(api *gin.RouterGroup, handler *Handler) {
	like := api.Group("/like")
	like.POST("/topic/:topic_id", handler.Like)
	like.DELETE("/topic/:topic_id", handler.Unlike)
	like.GET("/topic", handler.LikedTopics)
}

func registerTopicCollectionRoutes(api *gin.RouterGroup, handler *Handler) {
	collection := api.Group("/collection")
	collection.POST("/topic/:topic_id", handler.Collect)
	collection.DELETE("/topic/:topic_id", handler.Uncollect)
	collection.GET("/collection_topics", handler.CollectionTopics)
}
