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
	"github.com/Milchstrassse/Ecampus-go/internal/other/ad"
	"github.com/Milchstrassse/Ecampus-go/internal/other/merchant"
	"github.com/Milchstrassse/Ecampus-go/internal/other/notice"
	"github.com/Milchstrassse/Ecampus-go/internal/other/report"
	"github.com/Milchstrassse/Ecampus-go/internal/other/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/other/support"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type AdminHandlers struct {
	User      *user.AdminHandler
	Topic     *topic.AdminHandler
	Comment   *comment.AdminHandler
	Theme     *theme.AdminHandler
	File      *file.AdminHandler
	School    *school.AdminHandler
	Ad        *ad.AdminHandler
	Notice    *notice.AdminHandler
	Sensitive *sensitive.AdminHandler
	Report    *report.AdminHandler
	Support   *support.AdminHandler
	Merchant  *merchant.AdminHandler
	Event     *event.AdminHandler
	Monitor   *monitor.AdminHandler
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
	engine.POST("/admin/user/login", middleware.ControllerTimeTrack(rds, logger), handlers.User.Login)

	admin := engine.Group("/admin")
	admin.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.BlackListCheck(rds),
		middleware.RequestLog(logger),
		middleware.ControllerTimeTrack(rds, logger),
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

		admin.POST("/notice", handlers.Notice.NoticeAdd)
		admin.DELETE("/notice/:id", handlers.Notice.NoticeDelete)
		admin.PUT("/notice/:id", handlers.Notice.NoticeUpdate)
		admin.GET("/notice/:id", handlers.Notice.NoticeGet)
		admin.GET("/notice/list", handlers.Notice.NoticeList)

		admin.POST("/ad", handlers.Ad.AdAdd)
		admin.DELETE("/ad/:id", handlers.Ad.AdDelete)
		admin.PUT("/ad/:id", handlers.Ad.AdUpdate)
		admin.GET("/ad/:id", handlers.Ad.AdGet)
		admin.GET("/ad/list", handlers.Ad.AdList)

		admin.GET("/sensitive/getAllList", handlers.Sensitive.SensitiveGetAllList)
		admin.GET("/sensitive/getByWord", handlers.Sensitive.SensitiveGetByWord)
		admin.GET("/sensitive/getByWord/", handlers.Sensitive.SensitiveGetByWord)
		admin.DELETE("/sensitive/deleteByWord", handlers.Sensitive.SensitiveDeleteByWord)
		admin.DELETE("/sensitive/batchDelete", handlers.Sensitive.SensitiveBatchDelete)
		admin.POST("/sensitive/add", handlers.Sensitive.SensitiveAdd)
		admin.POST("/sensitive/batchAdd", handlers.Sensitive.SensitiveBatchAdd)
		admin.GET("/sensitive/page", handlers.Sensitive.SensitivePage)
		admin.GET("/sensitive/search_like", handlers.Sensitive.SensitiveSearchLike)
		admin.PUT("/sensitive/update", handlers.Sensitive.SensitiveUpdate)

		admin.PUT("/report_comment/:id", handlers.Report.ReportReview)
		admin.GET("/report_comment/list", handlers.Report.ReportList)

		admin.POST("/support", handlers.Support.SupportAdd)
		admin.PUT("/support", handlers.Support.SupportUpdate)
		admin.DELETE("/support/:id", handlers.Support.SupportDelete)
		admin.GET("/support/list", handlers.Support.SupportList)

		admin.POST("/merchant_theme", handlers.Merchant.MerchantAdd)
		admin.DELETE("/merchant_theme/:id", handlers.Merchant.MerchantDelete)
		admin.GET("/merchant_theme/get_all", handlers.Merchant.MerchantList)

		admin.POST("/task", handlers.Merchant.TaskAdd)
		admin.DELETE("/task/:id", handlers.Merchant.TaskDelete)
		admin.PUT("/task/:id", handlers.Merchant.TaskUpdate)
		admin.GET("/task/:id", handlers.Merchant.TaskGet)
		admin.GET("/task/list", handlers.Merchant.TaskList)

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
