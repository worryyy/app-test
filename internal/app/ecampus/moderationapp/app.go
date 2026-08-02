package moderationapp

import (
	"context"
	"os"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/app/moderationtarget"
	"github.com/Milchstrassse/Ecampus-go/internal/app/notificationadapter"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())

	svc := app.Moderation
	svc.SetTargetResolver(moderationtarget.Resolver{
		Users:       app.Users,
		Topics:      topic.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger),
		Comments:    comment.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger, nil),
		Chat:        chat.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger),
		Academic:    academic.NewService(app.Infra.MySQL, app.Infra.Logger),
		Marketplace: marketplace.NewService(app.Infra.MySQL, marketplace.NewGateway(os.Getenv("APP_PROFILE")), app.Infra.Logger),
	})
	svc.SetNotifier(notificationadapter.Adapter{Service: notification.NewService(app.Infra.Mongo, app.Infra.Redis, app.Infra.Logger)})
	moderation.RegisterProtectedRoutes(app.ProtectedAPI(), moderation.NewHandler(svc))

	return app.Run("ecampus-moderation")
}
