package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	engine.Use(middleware.CORS())

	pub := engine.Group("")
	{
		pub.POST("/api/user/login", handlers.User.Login)
		pub.POST("/api/user/refresh", handlers.User.RefreshToken)
		pub.PUT("/api/user/pre_authentication", handlers.User.PreAuth)

		pub.GET("/file/:md5", handlers.File.Download)
		pub.GET("/file", handlers.File.ListPublic)
	}

	api := engine.Group("/api")
	api.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.BlackListCheck(rds),
		middleware.RequestLog(logger),
		middleware.CertifiedUserCheck(db),
	)
	{
		api.GET("/user", handlers.User.GetCurrent)
		api.PUT("/user", handlers.User.Edit)
		api.GET("/user/nickname/random", handlers.User.RandomNickname)
		api.POST("/user/authentication", handlers.User.Authenticate)
		api.POST("/user/re_authentication", handlers.User.ReAuthenticate)
		api.POST("/user/del_authentication", handlers.User.DelAuthentication)
		api.POST("/user/check_login", handlers.User.CheckLogin)
		api.POST("/user/get_course_by_weeks", handlers.User.GetCourseByWeeks)
		api.POST("/user/get_exam", handlers.User.GetExam)
		api.POST("/user/get_exam_score", handlers.User.GetExamScore)
		api.GET("/user/user_profile", handlers.User.GetUserProfile)
		api.POST("/user/identity/anonymous", handlers.User.CreateAnonymous)
		api.PUT("/user/identity/anonymous/nickname", handlers.User.UpdateAnonymousNickname)
		api.GET("/user/identity/list", handlers.User.ListIdentity)
		api.POST("/user/identity/switch", handlers.User.SwitchIdentity)

		api.POST("/topic", handlers.Topic.Create)
		api.DELETE("/topic/:id", handlers.Topic.Delete)
		api.GET("/topic/:topic_id", handlers.Topic.GetByID)
		api.PUT("/topic/:topic_id", handlers.Topic.Update)
		api.GET("/topic/search", handlers.Topic.Search)
		api.GET("/topic", handlers.Topic.Mine)
		api.GET("/topic/theme", handlers.Topic.ThemeMine)
		api.GET("/topic/target_user_topics", handlers.Topic.TargetUserTopics)
		api.GET("/topic/follow_topics", handlers.Topic.FollowTopics)

		api.POST("/like/topic/:topic_id", handlers.Topic.Like)
		api.DELETE("/like/topic/:topic_id", handlers.Topic.Unlike)
		api.GET("/like/topic", handlers.Topic.LikedTopics)

		api.POST("/collection/topic/:topic_id", handlers.Topic.Collect)
		api.DELETE("/collection/topic/:topic_id", handlers.Topic.Uncollect)
		api.GET("/collection/collection_topics", handlers.Topic.CollectionTopics)

		api.POST("/comment/:topic_id", handlers.Comment.Create)
		api.DELETE("/comment/:topic_id/:comment_id", handlers.Comment.Delete)
		api.GET("/comment/:topic_id", handlers.Comment.ListByTopic)
		api.GET("/comment", handlers.Comment.Mine)
		api.GET("/comment/target_user_comments", handlers.Comment.TargetUserComments)
		api.POST("/comment_like/:comment_id", handlers.Comment.Like)
		api.DELETE("/comment_like/:comment_id", handlers.Comment.Unlike)

		api.GET("/conversation", handlers.Chat.Conversations)
		api.PUT("/conversation/conversation_enter", handlers.Chat.ConversationEnter)
		api.GET("/conversation/:id/unread_count", handlers.Chat.ConversationUnreadCount)
		api.GET("/conversation/conversation_query", handlers.Chat.ConversationQuery)
		api.GET("/conversation/profile_by_conversation_id", handlers.Chat.ProfileByConversationID)
		api.DELETE("/conversation/:id", handlers.Chat.DeleteConversation)
		api.GET("/message/:last_message_id", handlers.Chat.OfflineMessages)
		api.GET("/message/history_messages", handlers.Chat.HistoryMessages)
		api.GET("/message/unread_messages", handlers.Chat.UnreadMessages)
		api.GET("/notify", handlers.Chat.NotifyList)
		api.GET("/notify/:type/haveUnread", handlers.Chat.NotifyHaveUnread)
		api.GET("/notify/:type", handlers.Chat.NotifyLatest)


		api.POST("/course_color", handlers.School.CourseColor)
		api.GET("/term/list", handlers.School.TermList)
		api.GET("/term", handlers.School.CurrentTerm)

		api.POST("/theme/campus/init", handlers.Theme.InitCampusThemes)
		api.GET("/theme/campus", handlers.Theme.GetCampusThemes)
		api.POST("/wx/unlimited/wxa_code", handlers.User.UnlimitedWXACode)
	}

	fileAuth := engine.Group("/file")
	fileAuth.Use(middleware.JWTAuth(jwtHelper, rds))
	{
		fileAuth.POST("/upload", handlers.File.Upload)
		fileAuth.DELETE("/del/:md5", handlers.File.Delete)
	}

	engine.GET("/chat", handlers.Chat.WS)
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
