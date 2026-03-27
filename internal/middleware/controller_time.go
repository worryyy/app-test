package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/monitor"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

func ControllerTimeTrack(rds *redis.Client, logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		success, ok := c.Get("result_success")
		successFlag, ok := success.(bool)
		if !ok {
			return
		}

		record := monitor.ControllerTime{
			Controller: normalizeControllerName(c.HandlerName()),
			TimeCost:   time.Since(start).Milliseconds(),
			Success:    boolToInt(successFlag),
			AddTime:    time.Now(),
		}
		if rds == nil || record.Controller == "" {
			return
		}

		raw, err := json.Marshal(record)
		if err != nil {
			logger.Warn("marshal controller time failed", zap.Error(err))
			return
		}
		key := rediskey.ControllerTimePrefix + record.Controller
		if err := rds.LPush(c.Request.Context(), key, raw).Err(); err != nil {
			logger.Warn("push controller time failed", zap.Error(err), zap.String("key", key))
		}
	}
}

func normalizeControllerName(handlerName string) string {
	if handlerName == "" {
		return ""
	}
	short := handlerName[strings.LastIndex(handlerName, "/")+1:]
	short = strings.TrimSuffix(short, "-fm")
	short = strings.ReplaceAll(short, "(*", "")
	short = strings.ReplaceAll(short, ")", "")
	return short
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
