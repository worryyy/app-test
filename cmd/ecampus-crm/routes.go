package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
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

	}

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
}
