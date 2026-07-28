package ecampus

import (
	"context"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"go.uber.org/zap"
)

func startReservationJobs(ctx context.Context, service *reservation.Service, logger *zap.Logger) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := service.RunDueJobs(ctx); err != nil {
					logger.Warn("reservation due jobs failed", zap.Error(err))
				}
			}
		}
	}()
}

func startMarketplaceJobs(ctx context.Context, service *marketplace.Service, logger *zap.Logger) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := service.RunDueJobs(ctx); err != nil {
					logger.Warn("marketplace due jobs failed", zap.Error(err))
				}
				service.RetryPendingSettlements(ctx)
			}
		}
	}()
}
