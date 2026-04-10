package ecampuscrm

import (
	"context"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

func Run() error {
	infra, err := bootstrap.LoadInfrastructure(bootstrap.Options{
		ConfigDir:     "configs/ecampus-crm",
		SnowflakeNode: 1,
	})
	if err != nil {
		return err
	}
	defer infra.Close(context.Background())

	jwtHelper := jwtutil.NewHelper(infra.Config.JWT, infra.Redis)
	userSvc := user.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	schoolSvc := school.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, nil)
	sensitiveSvc := sensitive.NewService(infra.MySQL, infra.Logger)
	topicSvc := topic.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)

	engine := bootstrap.NewEngine()
	registerRoutes(engine, infra.Logger, infra.MySQL, jwtHelper, infra.Redis, adminHandlers{
		User:      user.NewAdminHandler(userSvc),
		School:    school.NewAdminHandler(schoolSvc),
		Sensitive: sensitive.NewAdminHandler(sensitiveSvc),
		Topic:     topic.NewAdminHandler(topicSvc),
	})

	server := bootstrap.NewHTTPServer(infra.Config.Server.Port, engine)
	return bootstrap.RunHTTPServer(server, infra.Logger, "ecampus-crm", 10*time.Second)
}
