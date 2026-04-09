package mq

import (
	"context"

	"go.uber.org/zap"
)

func (c *Consumers) filterSensitiveText(ctx context.Context, content string) (string, error) {
	if c == nil || c.filter == nil || content == "" {
		return content, nil
	}

	filtered, err := c.filter.FilterText(ctx, content)
	if err != nil {
		if c.logger != nil {
			c.logger.Warn("filter sensitive text failed", zap.Error(err))
		}
		return "", err
	}
	return filtered, nil
}
