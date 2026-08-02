package schoolapp

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())
	handler := school.NewHandler(school.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger))
	school.RegisterPublicRoutes(app.Engine, handler)
	school.RegisterProtectedRoutes(app.ProtectedAPI(), handler)
	return app.Run("ecampus-school")
}
