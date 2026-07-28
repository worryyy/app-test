package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

const authPermissionMsg = "当前接口需要认证用户权限"

const authAccountTypeAnonymous = "anonymous"

var (
	authPermissionResp = responses.New(false, http.StatusForbidden, authPermissionMsg)
	authInternalResp   = responses.New(false, http.StatusInternalServerError, "系统错误")
)

var authPermissionExcludes = map[string]struct{}{
	"/api/user/authentication":                {},
	"/api/user/login":                         {},
	"/api/user/refresh":                       {},
	"/api/user/pre_authentication":            {},
	"/api/notification/:id/read":              {},
	"/api/notification/read":                  {},
	"/api/moderation/reports/:id":             {},
	"/api/moderation/punishments/:id/appeals": {},
}

func CertifiedUserCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requiresCertifiedUser(c) {
			c.Next()
			return
		}

		claims := GetClaims(c)
		subjectID := certificationSubjectID(claims)
		if subjectID <= 0 {
			authPermissionResp.Resp(c)
			c.Abort()
			return
		}
		if db == nil {
			authInternalResp.Resp(c)
			c.Abort()
			return
		}

		var current user.User
		err := db.WithContext(c.Request.Context()).
			Select("stu_is_check, provisional_expires_at").
			Where("id = ?", subjectID).
			Take(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			authPermissionResp.Resp(c)
			c.Abort()
			return
		}
		if err != nil {
			authInternalResp.Resp(c)
			c.Abort()
			return
		}
		if !hasCertifiedAccess(current, time.Now()) {
			authPermissionResp.Resp(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func certificationSubjectID(claims *jwtutil.Claims) int64 {
	if claims == nil {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(claims.AccountType), authAccountTypeAnonymous) {
		return claims.RootUserID
	}
	return claims.UserID
}

func hasCertifiedAccess(current user.User, now time.Time) bool {
	if current.StuIsCheck {
		return true
	}
	return current.ProvisionalExpiresAt != nil && now.Before(*current.ProvisionalExpiresAt)
}

func requiresCertifiedUser(c *gin.Context) bool {
	method := c.Request.Method
	if method == http.MethodGet || method == http.MethodOptions {
		return false
	}

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	_, excluded := authPermissionExcludes[path]
	return !excluded
}
