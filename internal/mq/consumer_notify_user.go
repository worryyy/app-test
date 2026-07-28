package mq

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Consumers) handleNotifyUser(ctx context.Context, data json.RawMessage) error {
	var msg NotifyMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.TargetUserID == "" {
		return nil
	}

	if c.notificationWriter == nil {
		return fmt.Errorf("notification writer is not configured")
	}
	return c.notificationWriter.PersistLegacyNotification(ctx, msg)
}
