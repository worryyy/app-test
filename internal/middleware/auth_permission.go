package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

const authPermissionMsg = "当前接口需要进行认证后，方可使用"

var authPermissionExcludes = map[string]struct{}{
	"/api/user/authentication":     {},
	"/api/user/login":              {},
	"/api/user/refresh":            {},
	"/api/user/pre_authentication": {},
	"/admin/user/login":            {},
}

func CertifiedUserCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requiresCertifiedUser(c) {
			c.Next()
			return
		}

		claims := GetClaims(c)
		if claims == nil || claims.UserID <= 0 {
			result.Fail(c, result.CodeFail, authPermissionMsg)
			c.Abort()
			return
		}
		if db == nil {
			result.FailWithStatus(c, http.StatusInternalServerError, result.CodeUnknownError, "系统错误")
			c.Abort()
			return
		}

		var current user.User
		err := db.WithContext(c.Request.Context()).
			Select("stu_is_check").
			Where("id = ?", claims.UserID).
			Take(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Fail(c, result.CodeFail, authPermissionMsg)
			c.Abort()
			return
		}
		if err != nil {
			result.FailWithStatus(c, http.StatusInternalServerError, result.CodeUnknownError, "系统错误")
			c.Abort()
			return
		}
		if !current.StuIsCheck {
			result.Fail(c, result.CodeFail, authPermissionMsg)
			c.Abort()
			return
		}
		c.Next()
	}
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
