package chatapp

import (
	"context"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())

	svc := chat.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger)
	svc.SetCapabilityChecker(app.Capabilities)
	handler := chat.NewHandler(svc, app.Users, app.JWTHelper, app.Infra.Redis)
	chat.RegisterInfraRoutes(app.Engine, handler)
	chat.RegisterProtectedRoutes(app.ProtectedAPI(), handler)

	notificationSvc := notification.NewService(app.Infra.Mongo, app.Infra.Redis, app.Infra.Logger)
	notificationHandler := notification.NewHandler(notificationSvc, app.JWTHelper)
	notificationHandler.SetLegacyPusher(func(targetUserID string, payload any) error {
		return handler.PushNotification(context.Background(), targetUserID, payload)
	})
	closeSubscriber, err := notificationSvc.StartSubscriber(context.Background(), notificationHandler.Broadcast)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeSubscriber(); err != nil {
			app.Infra.Logger.Warn("close notification subscriber failed", zap.Error(err))
		}
	}()

	return app.Run("ecampus-chat")
}
