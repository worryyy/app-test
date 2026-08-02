package fileapp

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())
	handler := file.NewHandler(file.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger))
	file.RegisterPublicRoutes(app.Engine, handler)
	file.RegisterProtectedRoutes(app.FileAuthGroup(), handler)
	return app.Run("ecampus-file")
}
