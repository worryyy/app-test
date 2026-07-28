package ecampus

import (
	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/agentchat"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type userHandlers struct {
	User         *user.Handler
	Topic        *topic.Handler
	Comment      *comment.Handler
	Theme        *theme.Handler
	File         *file.Handler
	Chat         *chat.Handler
	Notification *notification.Handler
	Moderation   *moderation.Handler
	Academic     *academic.Handler
	Reservation  *reservation.Handler
	Marketplace  *marketplace.Handler
	School       *school.Handler
	Agent        *agentchat.Handler
}

func registerRoutes(
	engine *gin.Engine,
	logger *zap.Logger,
	db *gorm.DB,
	jwtHelper *jwtutil.Helper,
	rds *redis.Client,
	moderationSvc *moderation.Service,
	handlers userHandlers,
) {
	user.RegisterPublicRoutes(engine, handlers.User)
	school.RegisterPublicRoutes(engine, handlers.School)
	file.RegisterPublicRoutes(engine, handlers.File)
	chat.RegisterInfraRoutes(engine, handlers.Chat)
	notification.RegisterInfraRoutes(engine, handlers.Notification)
	agentchat.RegisterInfraRoutes(engine, handlers.Agent)

	api := engine.Group("/api")
	api.Use(
		middleware.JWTAuth(jwtHelper, rds),
		middleware.RequestLog(logger),
		moderation.AccountGuard(moderationSvc),
		middleware.CertifiedUserCheck(db),
	)

	user.RegisterProtectedRoutes(api, handlers.User)
	topic.RegisterProtectedRoutes(api, handlers.Topic)
	comment.RegisterProtectedRoutes(api, handlers.Comment)
	chat.RegisterProtectedRoutes(api, handlers.Chat)
	notification.RegisterProtectedRoutes(api, handlers.Notification)
	moderation.RegisterProtectedRoutes(api, handlers.Moderation)
	academic.RegisterProtectedRoutes(api, handlers.Academic)
	reservation.RegisterProtectedRoutes(api, handlers.Reservation)
	marketplace.RegisterProtectedRoutes(api, handlers.Marketplace)
	agentchat.RegisterProtectedRoutes(api, handlers.Agent)
	school.RegisterProtectedRoutes(api, handlers.School)
	theme.RegisterProtectedRoutes(api, handlers.Theme)

	fileAuth := engine.Group("/file")
	fileAuth.Use(middleware.JWTAuth(jwtHelper, rds))
	file.RegisterProtectedRoutes(fileAuth, handlers.File)
}
