package ecampuscrm

import (
	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/adminjwt"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type adminHandlers struct {
	User        *user.AdminHandler
	School      *school.AdminHandler
	Sensitive   *sensitive.AdminHandler
	Topic       *topic.AdminHandler
	Comment     *comment.AdminHandler
	Moderation  *moderation.AdminHandler
	Academic    *academic.AdminHandler
	Reservation *reservation.AdminHandler
	Marketplace *marketplace.AdminHandler
}

func registerRoutes(
	engine *gin.Engine,
	logger *zap.Logger,
	db *gorm.DB,
	adminHelper *adminjwt.Helper,
	rds *redis.Client,
	handlers adminHandlers,
) {
	user.RegisterAdminPublicRoutes(engine, handlers.User)

	admin := engine.Group("/admin")
	admin.Use(
		middleware.AdminJWTAuth(adminHelper, rds),
		middleware.RequestLog(logger),
		middleware.AdminPermissionCheck(db),
	)

	user.RegisterAdminRoutes(admin, handlers.User)
	school.RegisterAdminRoutes(admin, handlers.School)
	sensitive.RegisterAdminRoutes(admin, handlers.Sensitive)
	topic.RegisterAdminRoutes(admin, handlers.Topic)
	comment.RegisterAdminRoutes(admin, handlers.Comment)
	moderation.RegisterAdminRoutes(admin, handlers.Moderation)
	academic.RegisterAdminRoutes(admin, handlers.Academic)
	reservation.RegisterAdminRoutes(admin, handlers.Reservation)
	marketplace.RegisterAdminRoutes(admin, handlers.Marketplace)
}
