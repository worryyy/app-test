package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func BlackListCheck(rds *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		if rds == nil {
			c.Next()
			return
		}

		claims := GetClaims(c)
		if claims == nil {
			result.Fail(c, result.CodeForbidden, "权限不足")
			c.Abort()
			return
		}

		rootUserID := strconv.FormatInt(claims.RootUserID, 10)
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		blocked, err := rds.SIsMember(ctx, rediskey.GlobalBlacklist, rootUserID).Result()
		if err != nil {
			c.Next()
			return
		}
		if blocked {
			result.Fail(c, result.CodeForbidden, "权限不足")
			c.Abort()
			return
		}
		c.Next()
	}
}
