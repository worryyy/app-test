package themeapp

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())
	svc := theme.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger)
	theme.RegisterProtectedRoutes(app.ProtectedAPI(), theme.NewHandler(svc))
	return app.Run("ecampus-theme")
}
