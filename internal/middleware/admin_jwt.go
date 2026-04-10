package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/adminjwt"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const adminClaimsContextKey = "admin_claims"

var (
	adminAuthNotFoundResp   = responses.New(false, http.StatusUnauthorized, "authorization not found")
	adminTokenNotFoundResp  = responses.New(false, http.StatusUnauthorized, "admin token not existed or expired")
	adminTokenInvalidResp   = responses.New(false, http.StatusUnauthorized, "admin token invalid")
	adminSessionInvalidResp = responses.New(false, http.StatusUnauthorized, "admin session invalid")
)

func AdminJWTAuth(helper *adminjwt.Helper, rds *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if helper == nil {
			adminTokenInvalidResp.Resp(c)
			c.Abort()
			return
		}

		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if len(token) >= 7 && strings.EqualFold(token[:7], "Bearer ") {
			token = strings.TrimSpace(token[7:])
		}
		if token == "" {
			adminAuthNotFoundResp.Resp(c)
			c.Abort()
			return
		}

		claims, err := helper.ParseAndVerifyAccess(c.Request.Context(), token, rds)
		if err != nil {
			switch {
			case errors.Is(err, adminjwt.ErrTokenEmpty):
				adminAuthNotFoundResp.Resp(c)
			case errors.Is(err, adminjwt.ErrTokenNotExisted):
				adminTokenNotFoundResp.Resp(c)
			case errors.Is(err, adminjwt.ErrSessionInvalid):
				adminSessionInvalidResp.Resp(c)
			default:
				adminTokenInvalidResp.Resp(c)
			}
			c.Abort()
			return
		}

		c.Set(adminClaimsContextKey, claims)
		c.Next()
	}
}

func GetAdminClaims(c *gin.Context) *adminjwt.Claims {
	v, ok := c.Get(adminClaimsContextKey)
	if !ok || v == nil {
		return nil
	}
	claims, ok := v.(*adminjwt.Claims)
	if !ok {
		return nil
	}
	return claims
}
