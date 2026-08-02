package academicapp

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/app/useradapter"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())

	filter := sensitive.NewService(app.Infra.MySQL, app.Infra.Logger)
	defer filter.Close()
	svc := academic.NewService(app.Infra.MySQL, app.Infra.Logger)
	svc.SetProfileResolver(useradapter.Adapter{Service: app.Users})
	svc.SetCapabilityChecker(app.Capabilities)
	svc.SetSensitiveFilter(filter)
	svc.SetFileStore(file.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger))
	academic.RegisterProtectedRoutes(app.ProtectedAPI(), academic.NewHandler(svc))

	return app.Run("ecampus-academic")
}
