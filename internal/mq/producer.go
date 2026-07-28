package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/rediskey"
)

type BaseProducer struct {
	ch       *amqp.Channel
	exchange string
	routeKey string
	rds      *redis.Client
	logger   *zap.Logger
}

func NewBaseProducer(ch *amqp.Channel, exchange, routeKey string, rds *redis.Client, logger *zap.Logger) *BaseProducer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &BaseProducer{
		ch:       ch,
		exchange: exchange,
		routeKey: routeKey,
		rds:      rds,
		logger:   logger,
	}
}

func (p *BaseProducer) Send(ctx context.Context, data interface{}) error {
	if p == nil || p.ch == nil {
		return fmt.Errorf("producer not initialized")
	}

	uniqueID := time.Now().UnixNano()
	if p.rds != nil {
		id, err := p.rds.Incr(ctx, rediskey.MQUUIDKey).Result()
		if err != nil {
			return fmt.Errorf("increase mq uuid: %w", err)
		}
		uniqueID = id
	}

	body, err := json.Marshal(MQMessage{UniqueID: uniqueID, Data: data})
	if err != nil {
		return fmt.Errorf("marshal mq message: %w", err)
	}

	if err := p.ch.PublishWithContext(ctx, p.exchange, p.routeKey, true, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	}); err != nil {
		return fmt.Errorf("publish mq message: %w", err)
	}
	return nil
}

type Producer struct {
	ch       *amqp.Channel
	mongoDB  *mongo.Database
	logger   *zap.Logger
	rds      *redis.Client
	mu       sync.RWMutex
	producer map[string]*BaseProducer
}

func NewProducer(conn *amqp.Connection, rds *redis.Client, mongoDB *mongo.Database, logger *zap.Logger) (*Producer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if conn == nil {
		return nil, fmt.Errorf("rabbitmq connection is nil")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	if err := declareTopology(ch); err != nil {
		_ = ch.Close()
		return nil, err
	}

	p := &Producer{
		ch:      ch,
		mongoDB: mongoDB,
		logger:  logger,
		rds:     rds,
		producer: map[string]*BaseProducer{
			KeyTopicCheck:        NewBaseProducer(ch, Exchange, KeyTopicCheck, rds, logger),
			KeyAddComment:        NewBaseProducer(ch, Exchange, KeyAddComment, rds, logger),
			KeyNotifyUser:        NewBaseProducer(ch, Exchange, KeyNotifyUser, rds, logger),
			KeyUpdateTopicUser:   NewBaseProducer(ch, Exchange, KeyUpdateTopicUser, rds, logger),
			KeyUpdateCommentUser: NewBaseProducer(ch, Exchange, KeyUpdateCommentUser, rds, logger),
			KeyDeleteTopic:       NewBaseProducer(ch, Exchange, KeyDeleteTopic, rds, logger),
			KeyDeleteComment:     NewBaseProducer(ch, Exchange, KeyDeleteComment, rds, logger),
			KeyDie:               NewBaseProducer(ch, DieExchange, KeyDie, rds, logger),
		},
	}

	p.setupConfirmAndReturn()
	return p, nil
}

func (p *Producer) Close() error {
	if p == nil || p.ch == nil {
		return nil
	}
	return p.ch.Close()
}

func (p *Producer) get(key string) *BaseProducer {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.producer[key]
}

func (p *Producer) sendByKey(ctx context.Context, key string, data interface{}) error {
	base := p.get(key)
	if base == nil {
		return fmt.Errorf("mq producer key not found: %s", key)
	}
	return base.Send(ctx, data)
}

func (p *Producer) setupConfirmAndReturn() {
	if p.ch == nil {
		return
	}
	if err := p.ch.Confirm(false); err != nil {
		p.logger.Warn("enable rabbitmq confirm failed", zap.Error(err))
		return
	}

	confirms := p.ch.NotifyPublish(make(chan amqp.Confirmation, 128))
	returns := p.ch.NotifyReturn(make(chan amqp.Return, 128))

	go func() {
		for confirm := range confirms {
			if confirm.Ack {
				continue
			}
			p.saveMQLog(context.Background(), "to_broker_fail", map[string]interface{}{
				"deliveryTag": confirm.DeliveryTag,
				"ack":         confirm.Ack,
			})
		}
	}()

	go func() {
		for ret := range returns {
			p.saveMQLog(context.Background(), "to_queue_fail", map[string]interface{}{
				"exchange":   ret.Exchange,
				"routingKey": ret.RoutingKey,
				"replyCode":  ret.ReplyCode,
				"replyText":  ret.ReplyText,
				"body":       string(ret.Body),
			})
		}
	}()
}

func (p *Producer) saveMQLog(ctx context.Context, typ string, data interface{}) {
	if p.mongoDB == nil {
		p.logger.Warn("skip save mq log because mongo is nil", zap.String("type", typ))
		return
	}
	if _, err := p.mongoDB.Collection("campus_mq").InsertOne(ctx, MQLog{
		CreatedTime: time.Now(),
		Type:        typ,
		Data:        data,
	}); err != nil {
		p.logger.Warn("save mq log failed", zap.Error(err), zap.String("type", typ))
	}
}

func (p *Producer) SendTopicCheck(ctx context.Context, msg TopicCheckMsg) error {
	return p.sendByKey(ctx, KeyTopicCheck, msg)
}

func (p *Producer) SendAddComment(ctx context.Context, cmt comment.Comment) error {
	return p.sendByKey(ctx, KeyAddComment, buildAddCommentMsg(cmt))
}

func (p *Producer) SendNotifyUser(ctx context.Context, msg NotifyMsg) error {
	if msg.EventID <= 0 {
		msg.EventID = snowflake.Generate().Int64()
	}
	return p.sendByKey(ctx, KeyNotifyUser, msg)
}

func (p *Producer) SendNotify(ctx context.Context, msg NotifyMsg) error {
	return p.SendNotifyUser(ctx, msg)
}

func (p *Producer) SendUpdateTopicUser(ctx context.Context, msg TopicUserUpdateMsg) error {
	return p.sendByKey(ctx, KeyUpdateTopicUser, msg)
}

func (p *Producer) SendUpdateCommentUser(ctx context.Context, msg CommentUserUpdateMsg) error {
	return p.sendByKey(ctx, KeyUpdateCommentUser, msg)
}

func (p *Producer) SendDeleteTopic(ctx context.Context, msg TopicDeleteMsg) error {
	return p.sendByKey(ctx, KeyDeleteTopic, msg)
}

func (p *Producer) SendDeleteComment(ctx context.Context, topicID, commentID string) error {
	return p.sendByKey(ctx, KeyDeleteComment, CommentDeleteMsg{TopicID: topicID, CommentID: commentID})
}

func buildAddCommentMsg(cmt comment.Comment) AddCommentMsg {
	msg := AddCommentMsg{
		Comment: AddCommentPayload{
			ID:          cmt.ID,
			TopicID:     cmt.TopicID,
			Comment:     cmt.Comment,
			CreatedTime: cmt.CreatedTime,
			User:        buildAddCommentUser(cmt.User),
			ParentCmtID: cmt.ParentCmtID,
			RootCmtID:   cmt.RootCmtID,
			IsAuthor:    cmt.IsAuthor,
			LikeNum:     cmt.LikeNum,
			CommentNum:  cmt.CommentNum,
			HasCheck:    cmt.HasCheck,
		},
	}
	if cmt.Parent != nil {
		parent := buildAddCommentUser(*cmt.Parent)
		msg.Comment.Parent = &parent
	}
	return msg
}

func buildAddCommentUser(user comment.CommentUser) AddCommentUser {
	return AddCommentUser{
		UserID:      user.UserID,
		Avatar:      user.Avatar,
		NickName:    user.NickName,
		AccountType: user.AccountType,
		Signature:   user.Signature,
	}
}
