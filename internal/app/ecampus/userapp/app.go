package userapp

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusproducer"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

func Run() error {
	app, err := ecampusruntime.New(true)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())
	producer, err := ecampusproducer.New(app.Infra)
	if err != nil {
		return err
	}
	defer ecampusproducer.Close(app.Infra, producer)
	app.Users.SetProducer(producerAdapter{producer: producer})
	handler := user.NewHandler(app.Users)
	user.RegisterPublicRoutes(app.Engine, handler)
	user.RegisterProtectedRoutes(app.ProtectedAPI(), handler)
	return app.Run("ecampus-user")
}

type producerAdapter struct{ producer *mq.Producer }

func (a producerAdapter) SendTopicUserUpdate(ctx context.Context, msg user.TopicUserUpdateMsg) error {
	return a.producer.SendUpdateTopicUser(ctx, mq.TopicUserUpdateMsg(msg))
}
func (a producerAdapter) SendCommentUserUpdate(ctx context.Context, msg user.CommentUserUpdateMsg) error {
	return a.producer.SendUpdateCommentUser(ctx, mq.CommentUserUpdateMsg(msg))
}
