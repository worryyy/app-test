package marketplaceapp

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/app/notificationadapter"
	"github.com/Milchstrassse/Ecampus-go/internal/app/useradapter"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
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
	notifications := notification.NewService(app.Infra.Mongo, app.Infra.Redis, app.Infra.Logger)
	svc := marketplace.NewService(app.Infra.MySQL, marketplace.NewGateway(os.Getenv("APP_PROFILE")), app.Infra.Logger)
	svc.SetSellerVerifier(useradapter.Adapter{Service: app.Users})
	svc.SetCapabilityChecker(app.Capabilities)
	svc.SetSensitiveFilter(filter)
	svc.SetNotifier(notificationadapter.Adapter{Service: notifications})
	marketplace.RegisterProtectedRoutes(app.ProtectedAPI(), marketplace.NewHandler(svc))

	jobCtx, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	go runJobs(jobCtx, svc, app.Infra.Logger)
	return app.Run("ecampus-marketplace")
}

func runJobs(ctx context.Context, svc *marketplace.Service, logger *zap.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.RunDueJobs(ctx); err != nil {
				logger.Warn("marketplace due jobs failed", zap.Error(err))
			}
			svc.RetryPendingSettlements(ctx)
		}
	}
}
