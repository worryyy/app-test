package ecampuscrm

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type adminHandlers struct {
	User      *user.AdminHandler
	School    *school.AdminHandler
	Sensitive *sensitive.AdminHandler
	Topic     *topic.AdminHandler
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

	adminAuthOnly := engine.Group("/admin")
	adminAuthOnly.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.RequestLog(logger),
		middleware.AdminCheck(db),
	)
	user.RegisterAdminAuthOnlyRoutes(adminAuthOnly, handlers.User)

	admin := engine.Group("/admin")
	admin.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.RequestLog(logger),
		middleware.AdminCheck(db),
		middleware.CertifiedUserCheck(db),
	)

	user.RegisterAdminRoutes(admin, handlers.User)
	school.RegisterAdminRoutes(admin, handlers.School)
	sensitive.RegisterAdminRoutes(admin, handlers.Sensitive)
	topic.RegisterAdminRoutes(admin, handlers.Topic)
}
