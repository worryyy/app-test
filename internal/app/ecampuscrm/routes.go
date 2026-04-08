package ecampuscrm

import (
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
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type adminHandlers struct {
	User    *user.AdminHandler
	Comment *comment.AdminHandler
	Theme   *theme.AdminHandler
	File    *file.AdminHandler
	School  *school.AdminHandler
}

func registerRoutes(
	engine *gin.Engine,
	logger *zap.Logger,
	db *gorm.DB,
	jwtHelper *jwtutil.Helper,
	rds *redis.Client,
	handlers adminHandlers,
) {
	engine.POST("/admin/user/login", handlers.User.Login)

	admin := engine.Group("/admin")
	admin.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.RequestLog(logger),
		middleware.AdminCheck(db),
		middleware.CertifiedUserCheck(db),
	)

	user.RegisterAdminRoutes(admin, handlers.User)
	comment.RegisterAdminRoutes(admin, handlers.Comment)
	theme.RegisterAdminRoutes(admin, handlers.Theme)
	file.RegisterAdminRoutes(admin, handlers.File)
	school.RegisterAdminRoutes(admin, handlers.School)
}
