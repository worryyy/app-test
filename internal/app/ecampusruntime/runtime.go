package ecampusruntime

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/app/moderationadapter"
	"github.com/Milchstrassse/Ecampus-go/internal/app/useradapter"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

const (
	ConfigDir       = "configs/ecampus"
	ShutdownTimeout = 10 * time.Second
)

type App struct {
	Infra        *bootstrap.Infra
	JWTHelper    *jwtutil.Helper
	Engine       *gin.Engine
	Users        *user.Service
	Moderation   *moderation.Service
	Capabilities moderationadapter.Capability
}

func New(withRabbitMQ bool) (*App, error) {
	infra, err := bootstrap.LoadInfrastructure(bootstrap.Options{ConfigDir: ConfigDir, WithRabbitMQ: withRabbitMQ, SnowflakeNode: 1})
	if err != nil {
		return nil, err
	}
	users := user.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	moderationSvc := moderation.NewService(infra.MySQL, infra.Redis, infra.Logger)
	return &App{
		Infra: infra, JWTHelper: jwtutil.NewHelper(infra.Config.JWT, infra.Redis), Engine: bootstrap.NewEngine(),
		Users: users, Moderation: moderationSvc,
		Capabilities: moderationadapter.Capability{Moderation: moderationSvc, Users: useradapter.Adapter{Service: users}},
	}, nil
}

func (a *App) Close(ctx context.Context) {
	if a != nil && a.Infra != nil {
		a.Infra.Close(ctx)
	}
}

func (a *App) ProtectedAPI() *gin.RouterGroup {
	api := a.Engine.Group("/api")
	api.Use(
		middleware.JWTAuth(a.JWTHelper, a.Infra.Redis),
		middleware.RequestLog(a.Infra.Logger),
		moderation.AccountGuard(a.Moderation),
		middleware.CertifiedUserCheck(a.Infra.MySQL),
	)
	return api
}

func (a *App) FileAuthGroup() *gin.RouterGroup {
	group := a.Engine.Group("/file")
	group.Use(middleware.JWTAuth(a.JWTHelper, a.Infra.Redis))
	return group
}

func (a *App) Run(name string) error {
	server := bootstrap.NewHTTPServer(a.Infra.Config.Server.Port, a.Engine)
	return bootstrap.RunHTTPServer(server, a.Infra.Logger, name, ShutdownTimeout)
}
