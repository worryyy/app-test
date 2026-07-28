package ecampus

import (
	"context"
	"os"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
	"go.uber.org/zap"
)

func Run() error {
	infra, err := bootstrap.LoadInfrastructure(bootstrap.Options{
		ConfigDir:     "configs/ecampus",
		WithRabbitMQ:  true,
		SnowflakeNode: 1,
	})
	if err != nil {
		return err
	}
	defer infra.Close(context.Background())

	jwtHelper := jwtutil.NewHelper(infra.Config.JWT, infra.Redis)
	producer, err := mq.NewProducer(infra.RabbitMQ, infra.Redis, infra.Mongo, infra.Logger)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := producer.Close(); closeErr != nil {
			infra.Logger.Warn("close producer failed", zap.Error(closeErr))
		}
	}()

	userSvc := user.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	userSvc.SetProducer(newUserProducerAdapter(producer))
	topicSvc := topic.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	topicSvc.SetProducer(newTopicProducerAdapter(producer))
	commentSvc := comment.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, producer)
	sensitiveSvc := sensitive.NewService(infra.MySQL, infra.Logger)
	defer sensitiveSvc.Close()
	topicSvc.SetSensitiveFilter(sensitiveSvc)
	commentSvc.SetSensitiveFilter(sensitiveSvc)
	themeSvc := theme.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	fileSvc := file.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	chatSvc := chat.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	notificationSvc := notification.NewService(infra.Mongo, infra.Redis, infra.Logger)
	notificationSvc.SetIdentityResolver(notificationUserAdapter{service: userSvc})
	if err := notificationSvc.EnsureIndexes(context.Background()); err != nil {
		return err
	}
	moderationSvc := moderation.NewService(infra.MySQL, infra.Redis, infra.Logger)
	capabilityChecker := moderationCapabilityAdapter{moderation: moderationSvc, users: userSvc}
	academicSvc := academic.NewService(infra.MySQL, infra.Logger)
	academicSvc.SetProfileResolver(academicProfileAdapter{users: userSvc})
	academicSvc.SetCapabilityChecker(capabilityChecker)
	academicSvc.SetSensitiveFilter(sensitiveSvc)
	academicSvc.SetFileStore(fileSvc)
	reservationSvc := reservation.NewService(infra.MySQL, infra.Logger)
	reservationSvc.SetCapabilityChecker(capabilityChecker)
	reservationSvc.SetNotifier(reservationNotifierAdapter{service: notificationSvc})
	marketplaceSvc := marketplace.NewService(infra.MySQL, marketplace.NewGateway(os.Getenv("APP_PROFILE")), infra.Logger)
	marketplaceSvc.SetSellerVerifier(marketplaceSellerAdapter{users: userSvc})
	marketplaceSvc.SetCapabilityChecker(capabilityChecker)
	marketplaceSvc.SetSensitiveFilter(sensitiveSvc)
	marketplaceSvc.SetNotifier(marketplaceNotifierAdapter{service: notificationSvc})
	moderationSvc.SetTargetResolver(moderationTargetAdapter{users: userSvc, topics: topicSvc, comments: commentSvc, chat: chatSvc, academic: academicSvc, marketplace: marketplaceSvc})
	moderationSvc.SetNotifier(moderationNotifierAdapter{service: notificationSvc})
	topicSvc.SetCapabilityChecker(capabilityChecker)
	commentSvc.SetCapabilityChecker(capabilityChecker)
	chatSvc.SetCapabilityChecker(capabilityChecker)
	schoolSvc := school.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, producer)

	userH := user.NewHandler(userSvc)
	topicH := topic.NewHandler(topicSvc)
	commentH := comment.NewHandler(commentSvc)
	themeH := theme.NewHandler(themeSvc)
	fileH := file.NewHandler(fileSvc)
	chatH := chat.NewHandler(chatSvc, userSvc, jwtHelper, infra.Redis)
	notificationH := notification.NewHandler(notificationSvc, jwtHelper)
	moderationH := moderation.NewHandler(moderationSvc)
	academicH := academic.NewHandler(academicSvc)
	reservationH := reservation.NewHandler(reservationSvc)
	marketplaceH := marketplace.NewHandler(marketplaceSvc)
	notificationH.SetLegacyPusher(func(targetUserID string, payload any) error {
		return chatH.PushNotification(context.Background(), targetUserID, payload)
	})
	schoolH := school.NewHandler(schoolSvc)
	agentH, closeAgent, err := newAgentHandler(infra, jwtHelper)
	if err != nil {
		return err
	}
	defer closeAgent()

	engine := bootstrap.NewEngine()
	registerRoutes(engine, infra.Logger, infra.MySQL, jwtHelper, infra.Redis, moderationSvc, userHandlers{
		User:         userH,
		Topic:        topicH,
		Comment:      commentH,
		Theme:        themeH,
		File:         fileH,
		Chat:         chatH,
		Notification: notificationH,
		Moderation:   moderationH,
		Academic:     academicH,
		Reservation:  reservationH,
		Marketplace:  marketplaceH,
		School:       schoolH,
		Agent:        agentH,
	})
	closeNotificationSubscriber, err := notificationSvc.StartSubscriber(context.Background(), notificationH.Broadcast)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeNotificationSubscriber(); closeErr != nil {
			infra.Logger.Warn("close notification subscriber failed", zap.Error(closeErr))
		}
	}()
	jobContext, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	startReservationJobs(jobContext, reservationSvc, infra.Logger)
	startMarketplaceJobs(jobContext, marketplaceSvc, infra.Logger)

	consumers, err := mq.NewConsumers(infra.RabbitMQ, infra.Redis, infra.Mongo, infra.MySQL, infra.Config, infra.Logger, producer, sensitiveSvc)
	if err != nil {
		return err
	}
	consumers.SetNotificationWriter(notificationWriterAdapter{service: notificationSvc})
	if err := consumers.Start(); err != nil {
		_ = consumers.Close()
		return err
	}
	defer func() {
		if closeErr := consumers.Close(); closeErr != nil {
			infra.Logger.Warn("close consumers failed", zap.Error(closeErr))
		}
	}()

	server := bootstrap.NewHTTPServer(infra.Config.Server.Port, engine)
	return bootstrap.RunHTTPServer(server, infra.Logger, "ecampus", 10*time.Second)
}
