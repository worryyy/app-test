package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

var (
	authNotFoundResp  = responses.New(false,bizerr.CodeBizErr, "authorization 找不到", http.StatusUnauthorized)
	tokenNotFoundResp = responses.New(false,bizerr.CodeBizErr, "token 不存在,或已过期", http.StatusUnauthorized)
	tokenInvalidResp  = responses.New(false,bizerr.CodeBizErr, "token invalid", http.StatusUnauthorized)
)

func JWTAuth(helper *jwtutil.Helper, rds *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		if token == "" {
			authNotFoundResp.Resp(c)
			c.Abort()
			return
		}

		claims, err := helper.ParseAndVerify(c.Request.Context(), token, rds)
		if err != nil {
			switch {
			case errors.Is(err, jwtutil.ErrTokenEmpty):
				authNotFoundResp.Resp(c)
			case errors.Is(err, jwtutil.ErrTokenNotExisted):
				tokenNotFoundResp.Resp(c)
			default:
				tokenInvalidResp.Resp(c)
			}
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

func GetClaims(c *gin.Context) *jwtutil.Claims {
	v, ok := c.Get("claims")
	if !ok || v == nil {
		return nil
	}
	claims, ok := v.(*jwtutil.Claims)
	if !ok {
		return nil
	}
	return claims
}

func GetUserID(c *gin.Context) int64 {
	claims := GetClaims(c)
	if claims == nil {
		return 0
	}
	return claims.UserID
}
