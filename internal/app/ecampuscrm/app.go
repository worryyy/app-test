package ecampuscrm

import (
	"context"
	"os"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/adminjwt"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
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

	adminHelper := adminjwt.NewHelper(infra.Config.AdminJWT, infra.Redis)
	userSvc := user.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	schoolSvc := school.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, nil)
	sensitiveSvc := sensitive.NewService(infra.MySQL, infra.Logger)
	defer sensitiveSvc.Close()
	topicSvc := topic.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	commentSvc := comment.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, nil)
	notificationSvc := notification.NewService(infra.Mongo, infra.Redis, infra.Logger)
	if err := notificationSvc.EnsureIndexes(context.Background()); err != nil {
		return err
	}
	moderationSvc := moderation.NewService(infra.MySQL, infra.Redis, infra.Logger)
	academicSvc := academic.NewService(infra.MySQL, infra.Logger)
	reservationSvc := reservation.NewService(infra.MySQL, infra.Logger)
	reservationSvc.SetNotifier(reservationNotifierAdapter{notifications: notificationSvc})
	marketplaceSvc := marketplace.NewService(infra.MySQL, marketplace.NewGateway(os.Getenv("APP_PROFILE")), infra.Logger)
	marketplaceSvc.SetNotifier(marketplaceNotifierAdapter{notifications: notificationSvc})
	moderationSvc.SetTargetResolver(moderationTargetAdapter{users: userSvc, topics: topicSvc, comments: commentSvc, academic: academicSvc, marketplace: marketplaceSvc})
	moderationSvc.SetNotifier(moderationNotifierAdapter{notifications: notificationSvc})

	engine := bootstrap.NewEngine()
	registerRoutes(engine, infra.Logger, infra.MySQL, adminHelper, infra.Redis, adminHandlers{
		User:        user.NewAdminHandler(userSvc),
		School:      school.NewAdminHandler(schoolSvc),
		Sensitive:   sensitive.NewAdminHandler(sensitiveSvc),
		Topic:       topic.NewAdminHandler(topicSvc),
		Comment:     comment.NewAdminHandler(commentSvc),
		Moderation:  moderation.NewAdminHandler(moderationSvc),
		Academic:    academic.NewAdminHandler(academicSvc),
		Reservation: reservation.NewAdminHandler(reservationSvc),
		Marketplace: marketplace.NewAdminHandler(marketplaceSvc),
	})

	server := bootstrap.NewHTTPServer(infra.Config.Server.Port, engine)
	return bootstrap.RunHTTPServer(server, infra.Logger, "ecampus-crm", 10*time.Second)
}
