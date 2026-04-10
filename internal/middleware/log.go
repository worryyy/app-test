package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestLog(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := maskedRequestPath(c.Request.URL.Path, c.Request.URL.RawQuery)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("clientIP", c.ClientIP()),
		}
		if errMsg := requestErrorMessage(c); errMsg != "" {
			fields = append(fields, zap.String("error", errMsg))
		}

		if status >= 500 {
			logger.Error("http request", fields...)
			return
		}
		if status >= 400 {
			logger.Warn("http request", fields...)
			return
		}
		logger.Info("http request", fields...)
	}
}

func requestErrorMessage(c *gin.Context) string {
	if c == nil || len(c.Errors) == 0 {
		return ""
	}

	errs := make([]string, 0, len(c.Errors))
	for _, item := range c.Errors {
		if item == nil || item.Err == nil {
			continue
		}
		msg := strings.TrimSpace(item.Err.Error())
		if msg == "" {
			continue
		}
		errs = append(errs, msg)
	}
	return strings.Join(errs, " | ")
}

func maskedRequestPath(path, rawQuery string) string {
	if rawQuery == "" {
		return path
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return path
	}

	for key := range values {
		if !isSensitiveQueryKey(key) {
			continue
		}
		values[key] = []string{"***"}
	}

	encoded := values.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func isSensitiveQueryKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "pwd", "password", "token", "refresh_token", "refreshtoken", "secondary_password", "secondarypassword":
		return true
	default:
		return false
	}
}
