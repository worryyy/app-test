package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/event"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/monitor"
	"github.com/Milchstrassse/Ecampus-go/internal/other"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type AdminHandlers struct {
	User    *user.AdminHandler
	Topic   *topic.AdminHandler
	Comment *comment.AdminHandler
	Theme   *theme.AdminHandler
	File    *file.AdminHandler
	School  *school.AdminHandler
	Other   *other.AdminHandler
	Event   *event.AdminHandler
	Monitor *monitor.AdminHandler
}

func registerAdminRoutes(
	engine *gin.Engine,
	logger *zap.Logger,
	db *gorm.DB,
	jwtHelper *jwtutil.Helper,
	rds *redis.Client,
	handlers AdminHandlers,
) {
	engine.Use(middleware.CORS())
	engine.POST("/admin/user/login", handlers.User.Login)

	admin := engine.Group("/admin")
	admin.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.BlackListCheck(rds),
		middleware.RequestLog(logger),
		middleware.AdminCheck(db),
		middleware.CertifiedUserCheck(db),
	)
	{
		admin.POST("/user", handlers.User.AddUser)
		admin.POST("/user/add", handlers.User.AddAdmin)
		admin.DELETE("/user/:id", handlers.User.DeleteUser)
		admin.PUT("/user/:id", handlers.User.EditUser)
		admin.GET("/user/:id", handlers.User.GetUser)
		admin.GET("/user/list", handlers.User.ListUsers)
		admin.POST("/user/clear", handlers.User.ClearAuthentication)
		admin.POST("/user/course", handlers.User.UserCourse)
		admin.POST("/user/add_black_list", handlers.User.AddBlackList)
		admin.DELETE("/user/del_black_list", handlers.User.DelBlackList)
		admin.GET("/user/black_list", handlers.User.BlackList)
		admin.GET("/user/certification/list", handlers.User.CertificationList)
		admin.POST("/user/certification/review", handlers.User.CertificationReview)

		admin.DELETE("/topic/:topic_id", handlers.Topic.Delete)
		admin.GET("/topic/refresh_suggest", handlers.Topic.RefreshSuggest)
		admin.DELETE("/comment/:topic_id/:comment_id", handlers.Comment.Delete)

		admin.PUT("/theme/:id", handlers.Theme.Update)
		admin.GET("/theme", handlers.Theme.List)
		admin.PUT("/theme/search", handlers.Theme.UpdateSearch)
		admin.POST("/theme/suggest", handlers.Theme.UpdateSuggest)
		admin.POST("/theme/campus", handlers.Theme.AddCampusTheme)
		admin.DELETE("/theme/campus/:themeId", handlers.Theme.DeleteCampusTheme)

		admin.POST("/file", handlers.File.SetPublic)
		admin.GET("/file", handlers.File.List)

		admin.POST("/term", handlers.School.AddTerm)
		admin.DELETE("/term/:id", handlers.School.DeleteTerm)
		admin.POST("/term/cur", handlers.School.SetCurrentTerm)

		admin.POST("/notice", handlers.Other.NoticeAdd)
		admin.DELETE("/notice/:id", handlers.Other.NoticeDelete)
		admin.PUT("/notice/:id", handlers.Other.NoticeUpdate)
		admin.GET("/notice/:id", handlers.Other.NoticeGet)
		admin.GET("/notice/list", handlers.Other.NoticeList)

		admin.POST("/ad", handlers.Other.AdAdd)
		admin.DELETE("/ad/:id", handlers.Other.AdDelete)
		admin.PUT("/ad/:id", handlers.Other.AdUpdate)
		admin.GET("/ad/:id", handlers.Other.AdGet)
		admin.GET("/ad/list", handlers.Other.AdList)

		admin.GET("/sensitive/getAllList", handlers.Other.SensitiveGetAllList)
		admin.GET("/sensitive/getByWord", handlers.Other.SensitiveGetByWord)
		admin.DELETE("/sensitive/deleteByWord", handlers.Other.SensitiveDeleteByWord)
		admin.DELETE("/sensitive/batchDelete", handlers.Other.SensitiveBatchDelete)
		admin.POST("/sensitive/add", handlers.Other.SensitiveAdd)
		admin.POST("/sensitive/batchAdd", handlers.Other.SensitiveBatchAdd)
		admin.GET("/sensitive/page", handlers.Other.SensitivePage)
		admin.GET("/sensitive/search_like", handlers.Other.SensitiveSearchLike)
		admin.PUT("/sensitive/update", handlers.Other.SensitiveUpdate)

		admin.PUT("/report_comment/:id", handlers.Other.ReportReview)
		admin.GET("/report_comment/list", handlers.Other.ReportList)

		admin.POST("/support", handlers.Other.SupportAdd)
		admin.PUT("/support", handlers.Other.SupportUpdate)
		admin.DELETE("/support/:id", handlers.Other.SupportDelete)
		admin.GET("/support/list", handlers.Other.SupportList)

		admin.POST("/merchant_theme", handlers.Other.MerchantAdd)
		admin.DELETE("/merchant_theme/:id", handlers.Other.MerchantDelete)
		admin.GET("/merchant_theme/get_all", handlers.Other.MerchantList)

		admin.POST("/task", handlers.Other.TaskAdd)
		admin.DELETE("/task/:id", handlers.Other.TaskDelete)
		admin.PUT("/task/:id", handlers.Other.TaskUpdate)
		admin.GET("/task/:id", handlers.Other.TaskGet)
		admin.GET("/task/list", handlers.Other.TaskList)

		admin.DELETE("/event/:id", handlers.Event.Delete)
		admin.PUT("/event/:id", handlers.Event.Update)
		admin.GET("/event/:id", handlers.Event.Get)
		admin.GET("/event/list", handlers.Event.List)

		admin.GET("/local_cache/all_key", handlers.Monitor.CacheNames)
		admin.GET("/local_cache/stats", handlers.Monitor.CacheStats)
	}

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
}
