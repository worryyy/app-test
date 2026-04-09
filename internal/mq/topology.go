package mq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func declareTopology(ch *amqp.Channel) error {
	if ch == nil {
		return fmt.Errorf("rabbitmq channel is nil")
	}

	if err := ch.ExchangeDeclare(Exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", Exchange, err)
	}
	if err := ch.ExchangeDeclare(DieExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", DieExchange, err)
	}

	deadArgs := amqp.Table{
		"x-dead-letter-exchange":    DieExchange,
		"x-dead-letter-routing-key": KeyDie,
	}
	if err := declareQueueAndBind(ch, QueueCommentUpdate, Exchange, KeyUpdateCommentUser, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueTopicUpdate, Exchange, KeyUpdateTopicUser, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueTopicDelete, Exchange, KeyDeleteTopic, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueCommentDelete, Exchange, KeyDeleteComment, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueCommentAdd, Exchange, KeyAddComment, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueTopicCheck, Exchange, KeyTopicCheck, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueNotifyUser, Exchange, KeyNotifyUser, deadArgs); err != nil {
		return err
	}
	if err := declareQueueAndBind(ch, QueueDie, DieExchange, KeyDie, nil); err != nil {
		return err
	}
	return nil
}

func declareQueueAndBind(
	ch *amqp.Channel,
	queueName string,
	exchange string,
	routingKey string,
	args amqp.Table,
) error {
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue %s: %w", queueName, err)
	}
	if err := ch.QueueBind(queueName, routingKey, exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue %s: %w", queueName, err)
	}
	return nil
}
