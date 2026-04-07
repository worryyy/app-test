package main

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type UserHandlers struct {
	User    *user.Handler
	Topic   *topic.Handler
	Comment *comment.Handler
	Theme   *theme.Handler
	File    *file.Handler
	Chat    *chat.Handler
	School  *school.Handler
}

func registerUserRoutes(
	engine *gin.Engine,
	logger *zap.Logger,
	db *gorm.DB,
	jwtHelper *jwtutil.Helper,
	rds *redis.Client,
	handlers UserHandlers,
) {
	registerUserPublicRoutes(engine, handlers)
	registerSchoolPublicRoutes(engine, handlers.School)
	registerFileAuthRoutes(engine, jwtHelper, rds, handlers.File)
	registerUserInfraRoutes(engine, handlers)

	api := engine.Group("/api")
	api.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.RequestLog(logger),
		middleware.CertifiedUserCheck(db),
	)

	registerUserProfileRoutes(api, handlers.User)
	registerUserFollowRoutes(api, handlers.User)
	registerUserIdentityRoutes(api, handlers.User)
	registerTopicRoutes(api, handlers.Topic)
	registerTopicLikeRoutes(api, handlers.Topic)
	registerTopicCollectionRoutes(api, handlers.Topic)
	registerCommentRoutes(api, handlers.Comment)
	registerChatRoutes(api, handlers.Chat)
	registerSchoolRoutes(api, handlers.School)
	registerThemeRoutes(api, handlers.Theme)
	registerWXRoutes(api, handlers.User)
}

func registerUserPublicRoutes(engine *gin.Engine, handlers UserHandlers) {
	pub := engine.Group("")
	{
		pub.POST("/api/user/login", handlers.User.Login)
		pub.POST("/api/user/refresh", handlers.User.RefreshToken)
		pub.PUT("/api/user/pre_authentication", handlers.User.PreAuth)

		pub.GET("/file/:md5", handlers.File.Download)
		pub.GET("/file", handlers.File.ListPublic)
	}
}

func registerSchoolPublicRoutes(engine *gin.Engine, handler *school.Handler) {
	pub := engine.Group("")
	{
		pub.GET("/api/term", handler.CurTerm)
	}
}

func registerFileAuthRoutes(
	engine *gin.Engine,
	jwtHelper *jwtutil.Helper,
	rds *redis.Client,
	handler *file.Handler,
) {
	fileAuth := engine.Group("/file")
	fileAuth.Use(middleware.JWTAuth(jwtHelper, rds))
	{
		fileAuth.POST("/upload", handler.Upload)
		fileAuth.DELETE("/del/:md5", handler.Delete)
	}
}

func registerUserInfraRoutes(engine *gin.Engine, handlers UserHandlers) {
	engine.GET("/chat", handlers.Chat.WS)
}

func registerUserProfileRoutes(api *gin.RouterGroup, handler *user.Handler) {
	api.GET("/user", handler.GetCurrent)
	api.PUT("/user", handler.Edit)
	api.GET("/user/nickname/random", handler.RandomNickname)
	api.GET("/user/user_profile", handler.GetUserProfile)

}

func registerUserIdentityRoutes(api *gin.RouterGroup, handler *user.Handler) {
	api.POST("/user/identity/anonymous", handler.CreateAnonymous)
	api.PUT("/user/identity/anonymous/nickname", handler.UpdateAnonymousNickname)
	api.GET("/user/identity/list", handler.ListIdentity)
	api.POST("/user/identity/switch", handler.SwitchIdentity)
}

func registerUserFollowRoutes(api *gin.RouterGroup, handler *user.Handler) {
	api.POST("/user/follow", handler.Follow)
	api.DELETE("/user/follow", handler.Unfollow)
	api.GET("/user/stats", handler.GetUserStats)
}

func registerTopicRoutes(api *gin.RouterGroup, handler *topic.Handler) {
	api.POST("/topic", handler.Create)
	api.DELETE("/topic/:id", handler.Delete)
	api.GET("/topic/:topic_id", handler.GetByID)
	api.PUT("/topic/:topic_id", handler.Update)
	api.GET("/topic/search", handler.Search)
	api.GET("/topic", handler.Mine)
	api.GET("/topic/target_user_topics", handler.TargetUserTopics)
}

func registerTopicLikeRoutes(api *gin.RouterGroup, handler *topic.Handler) {
	api.POST("/like/topic/:topic_id", handler.Like)
	api.DELETE("/like/topic/:topic_id", handler.Unlike)
	api.GET("/like/topic", handler.LikedTopics)
}

func registerTopicCollectionRoutes(api *gin.RouterGroup, handler *topic.Handler) {
	api.POST("/collection/topic/:topic_id", handler.Collect)
	api.DELETE("/collection/topic/:topic_id", handler.Uncollect)
	api.GET("/collection/collection_topics", handler.CollectionTopics)
}

func registerCommentRoutes(api *gin.RouterGroup, handler *comment.Handler) {
	api.POST("/comment/:topic_id", handler.Create)
	api.DELETE("/comment/:topic_id/:comment_id", handler.Delete)
	api.GET("/comment/:topic_id", handler.ListByTopic)
	api.GET("/comment", handler.Mine)
	api.GET("/comment/target_user_comments", handler.TargetUserComments)
	api.POST("/comment_like/:comment_id", handler.Like)
	api.DELETE("/comment_like/:comment_id", handler.Unlike)
}

func registerChatRoutes(api *gin.RouterGroup, handler *chat.Handler) {
	api.GET("/conversation", handler.Conversations)
	api.PUT("/conversation/conversation_enter", handler.ConversationEnter)
	api.GET("/conversation/:id/unread_count", handler.ConversationUnreadCount)
	api.GET("/conversation/conversation_query", handler.ConversationQuery)
	api.GET("/conversation/profile_by_conversation_id", handler.ProfileByConversationID)
	api.DELETE("/conversation/:id", handler.DeleteConversation)
	api.GET("/message/:last_message_id", handler.OfflineMessages)
	api.GET("/message/history_messages", handler.HistoryMessages)
	api.GET("/message/unread_messages", handler.UnreadMessages)
	api.GET("/notify", handler.NotifyList)
	api.GET("/notify/:type/haveUnread", handler.NotifyHaveUnread)
	api.GET("/notify/:type", handler.NotifyLatest)
}

func registerSchoolRoutes(api *gin.RouterGroup, handler *school.Handler) {
	api.POST("/user/authentication", handler.Authenticate)
	api.POST("/user/re_authentication", handler.ReAuthenticate)
	api.POST("/user/get_course_by_weeks", handler.GetCourseByWeeks)
	api.POST("/user/get_exam", handler.GetExam)
	api.POST("/user/get_exam_score", handler.GetExamScore)
}

func registerThemeRoutes(api *gin.RouterGroup, handler *theme.Handler) {
	api.POST("/theme/campus/init", handler.InitCampusThemes)
	api.GET("/theme/campus", handler.GetCampusThemes)
}

func registerWXRoutes(api *gin.RouterGroup, handler *user.Handler) {
	api.POST("/wx/unlimited/wxa_code", handler.UnlimitedWXACode)
}
