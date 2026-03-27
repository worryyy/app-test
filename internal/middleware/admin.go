package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

func AdminCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			result.Fail(c, result.CodeForbidden, "权限不足")
			c.Abort()
			return
		}

		isAdminToken := claims.Power > 0 && ((claims.Power>>1)&1) == 1

		var count int64
		if db == nil {
			result.Fail(c, result.CodeForbidden, "权限不足")
			c.Abort()
			return
		}

		if err := db.Model(&user.Admin{}).Where("user_id = ?", claims.UserID).Count(&count).Error; err != nil {
			result.Fail(c, result.CodeForbidden, "权限不足")
			c.Abort()
			return
		}
		isAdminUser := count > 0

		if !isAdminToken || !isAdminUser {
			result.Fail(c, result.CodeForbidden, "权限不足")
			c.Abort()
			return
		}
		c.Next()
	}
}
