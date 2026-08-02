package reservationapp

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/app/notificationadapter"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())

	notifications := notification.NewService(app.Infra.Mongo, app.Infra.Redis, app.Infra.Logger)
	svc := reservation.NewService(app.Infra.MySQL, app.Infra.Logger)
	svc.SetCapabilityChecker(app.Capabilities)
	svc.SetNotifier(notificationadapter.Adapter{Service: notifications})
	reservation.RegisterProtectedRoutes(app.ProtectedAPI(), reservation.NewHandler(svc))

	jobCtx, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	go runJobs(jobCtx, svc, app.Infra.Logger)
	return app.Run("ecampus-reservation")
}

func runJobs(ctx context.Context, svc *reservation.Service, logger *zap.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.RunDueJobs(ctx); err != nil {
				logger.Warn("reservation due jobs failed", zap.Error(err))
			}
		}
	}
}
