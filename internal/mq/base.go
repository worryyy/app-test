package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func HandleWithDedup(
	rds *redis.Client,
	prefixKey string,
	delivery amqp.Delivery,
	handler func(ctx context.Context, data json.RawMessage) error,
	logger *zap.Logger,
) {
	if logger == nil {
		logger = zap.NewNop()
	}
	ctx := context.Background()

	message, err := decodeMQMessage(delivery.Body)
	if err != nil {
		logger.Error("decode mq message failed", zap.Error(err))
		_ = delivery.Nack(false, false)
		return
	}

	if rds != nil {
		dedupKey := prefixKey + strconv.FormatInt(message.UniqueID, 10)
		status, statusErr := rds.Get(ctx, dedupKey).Int()
		if statusErr == nil && status == MsgPost {
			_ = delivery.Ack(false)
			return
		}
		if statusErr == nil && status == MsgIng {
			_ = delivery.Ack(false)
			return
		}
		if setErr := rds.Set(ctx, dedupKey, MsgIng, 3*24*time.Hour).Err(); setErr != nil {
			logger.Warn("set mq dedup ing failed", zap.Error(setErr), zap.String("key", dedupKey))
		}
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		if err := handler(ctx, message.Data); err == nil {
			_ = delivery.Ack(false)
			setDedupDone(ctx, rds, prefixKey, message.UniqueID, logger)
			return
		} else {
			lastErr = err
			logger.Error("mq consumer retry", zap.Int("attempt", i+1), zap.Error(err))
		}
	}

	logger.Error("mq consumer failed", zap.Error(lastErr))
	_ = delivery.Nack(false, false)
}

func setDedupDone(ctx context.Context, rds *redis.Client, prefixKey string, uniqueID int64, logger *zap.Logger) {
	if rds == nil {
		return
	}
	dedupKey := prefixKey + strconv.FormatInt(uniqueID, 10)
	if err := rds.Set(ctx, dedupKey, MsgPost, 3*24*time.Hour).Err(); err != nil {
		logger.Warn("set mq dedup done failed", zap.Error(err), zap.String("key", dedupKey))
	}
}

func decodeMQMessage(body []byte) (*rawMQMessage, error) {
	var message rawMQMessage
	if err := json.Unmarshal(body, &message); err != nil {
		return nil, fmt.Errorf("unmarshal mq envelope: %w", err)
	}
	if len(message.Data) == 0 {
		message.Data = []byte("{}")
	}
	return &message, nil
}

func decodeData(raw json.RawMessage, out interface{}) error {
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unmarshal mq data: %w", err)
	}
	return nil
}

type rawMQMessage struct {
	UniqueID int64           `json:"uniqueId"`
	Data     json.RawMessage `json:"data"`
}
