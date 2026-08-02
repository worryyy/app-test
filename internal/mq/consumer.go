package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/integration/wxutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
)

type Consumers struct {
	ch       *amqp.Channel
	rds      *redis.Client
	mongoDB  *mongo.Database
	db       *gorm.DB
	cfg      *config.Config
	logger   *zap.Logger
	producer *Producer
	wxClient *wxutil.Client
	filter   sensitive.Filter

	notificationWriter NotificationWriter
	wg                 sync.WaitGroup
	stopCh             chan struct{}
}

func NewConsumers(
	conn *amqp.Connection,
	rds *redis.Client,
	mongoDB *mongo.Database,
	db *gorm.DB,
	cfg *config.Config,
	logger *zap.Logger,
	producer *Producer,
	filter sensitive.Filter,
) (*Consumers, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conn == nil {
		return nil, fmt.Errorf("rabbitmq connection is nil")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open rabbitmq consumer channel: %w", err)
	}
	if err := declareTopology(ch); err != nil {
		_ = ch.Close()
		return nil, err
	}

	consumers := &Consumers{
		ch:       ch,
		rds:      rds,
		mongoDB:  mongoDB,
		db:       db,
		cfg:      cfg,
		logger:   logger,
		producer: producer,
		filter:   filter,
		stopCh:   make(chan struct{}),
	}
	if cfg != nil {
		consumers.wxClient = wxutil.NewClient(cfg.WX, logger)
	}
	return consumers, nil
}

type NotificationWriter interface {
	PersistLegacyNotification(ctx context.Context, msg NotifyMsg) error
}

func (c *Consumers) SetNotificationWriter(writer NotificationWriter) {
	c.notificationWriter = writer
}

func (c *Consumers) Start() error {
	if c.ch == nil {
		return fmt.Errorf("rabbitmq consumer channel is nil")
	}

	handlers := map[string]func(ctx context.Context, data json.RawMessage) error{
		QueueTopicCheck:    c.handleTopicCheck,
		QueueCommentAdd:    c.handleCommentAdd,
		QueueTopicUpdate:   c.handleTopicUpdate,
		QueueTopicDelete:   c.handleTopicDelete,
		QueueCommentUpdate: c.handleCommentUpdate,
		QueueCommentDelete: c.handleCommentDelete,
		QueueNotifyUser:    c.handleNotifyUser,
		QueueDie:           c.handleDie,
	}

	return c.startHandlers(handlers)
}

func (c *Consumers) StartNotification() error {
	if c.ch == nil {
		return fmt.Errorf("rabbitmq consumer channel is nil")
	}
	return c.startHandlers(c.notificationHandlers())
}

func (c *Consumers) StartTopic() error {
	if c.ch == nil {
		return fmt.Errorf("rabbitmq consumer channel is nil")
	}
	return c.startHandlers(c.topicHandlers())
}

func (c *Consumers) topicHandlers() map[string]func(context.Context, json.RawMessage) error {
	return map[string]func(context.Context, json.RawMessage) error{
		QueueTopicCheck:  c.handleTopicCheck,
		QueueTopicUpdate: c.handleTopicUpdate,
		QueueTopicDelete: c.handleTopicDelete,
	}
}

func (c *Consumers) StartComment() error {
	if c.ch == nil {
		return fmt.Errorf("rabbitmq consumer channel is nil")
	}
	return c.startHandlers(c.commentHandlers())
}

func (c *Consumers) commentHandlers() map[string]func(context.Context, json.RawMessage) error {
	return map[string]func(context.Context, json.RawMessage) error{
		QueueCommentAdd:    c.handleCommentAdd,
		QueueCommentUpdate: c.handleCommentUpdate,
		QueueCommentDelete: c.handleCommentDelete,
	}
}

func (c *Consumers) StartDeadLetters() error {
	if c.ch == nil {
		return fmt.Errorf("rabbitmq consumer channel is nil")
	}
	return c.startHandlers(c.deadLetterHandlers())
}

func (c *Consumers) notificationHandlers() map[string]func(context.Context, json.RawMessage) error {
	return map[string]func(context.Context, json.RawMessage) error{QueueNotifyUser: c.handleNotifyUser}
}

func (c *Consumers) deadLetterHandlers() map[string]func(context.Context, json.RawMessage) error {
	return map[string]func(context.Context, json.RawMessage) error{QueueDie: c.handleDie}
}

func (c *Consumers) startHandlers(handlers map[string]func(ctx context.Context, data json.RawMessage) error) error {
	for queue, handler := range handlers {
		msgs, err := c.ch.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("consume queue %s: %w", queue, err)
		}

		q := queue
		h := handler
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			for {
				select {
				case <-c.stopCh:
					return
				case msg, ok := <-msgs:
					if !ok {
						return
					}
					HandleWithDedup(c.rds, dedupPrefix(q), msg, h, c.logger)
				}
			}
		}()
	}
	return nil
}

func (c *Consumers) Close() error {
	close(c.stopCh)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		c.logger.Warn("mq consumers close timeout")
	}

	if c.ch != nil {
		return c.ch.Close()
	}
	return nil
}

func (c *Consumers) sendNotify(ctx context.Context, msg NotifyMsg) {
	if c.producer == nil {
		return
	}
	if err := c.producer.SendNotifyUser(ctx, msg); err != nil {
		c.logger.Warn("send notify mq failed", zap.Error(err), zap.String("targetUserID", msg.TargetUserID))
	}
}

func dedupPrefix(queue string) string {
	switch queue {
	case QueueTopicCheck:
		return rediskey.TopicCreateCache
	case QueueCommentAdd:
		return rediskey.AddMsgCache
	case QueueTopicUpdate, QueueCommentUpdate:
		return rediskey.UpdateMsgCache
	case QueueTopicDelete, QueueCommentDelete:
		return rediskey.DeleteMsgCache
	case QueueNotifyUser:
		return rediskey.NotifyCache
	case QueueDie:
		return rediskey.DeleteMsgCache
	default:
		return "campus:mq:dedup:"
	}
}
