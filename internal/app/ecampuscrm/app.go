package ecampuscrm

import (
	"context"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
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
	commentSvc := comment.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, nil)
	themeSvc := theme.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	fileSvc := file.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	schoolSvc := school.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, nil)

	engine := bootstrap.NewEngine()
	registerRoutes(engine, infra.Logger, infra.MySQL, jwtHelper, infra.Redis, adminHandlers{
		User:    user.NewAdminHandler(userSvc),
		Comment: comment.NewAdminHandler(commentSvc),
		Theme:   theme.NewAdminHandler(themeSvc),
		File:    file.NewAdminHandler(fileSvc),
		School:  school.NewAdminHandler(schoolSvc),
	})

	server := bootstrap.NewHTTPServer(infra.Config.Server.Port, engine)
	return bootstrap.RunHTTPServer(server, infra.Logger, "ecampus-crm", 10*time.Second)
}
