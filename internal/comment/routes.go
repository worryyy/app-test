package comment

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	comment := api.Group("/comment")
	comment.POST("/:topic_id", handler.Create)
	comment.DELETE("/:topic_id/:comment_id", handler.Delete)
	comment.GET("/:topic_id", handler.ListByTopic)
	comment.GET("", handler.Mine)
	comment.GET("/target_user_comments", handler.TargetUserComments)
	comment.POST("/like/:comment_id", handler.Like)
	comment.DELETE("/like/:comment_id", handler.Unlike)
}
