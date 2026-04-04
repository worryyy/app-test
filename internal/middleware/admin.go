package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

var adminForbiddenResp = responses.New(false,bizerr.CodeBizErr, "权限不足", http.StatusForbidden)

func AdminCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}

		isAdminToken := claims.Power > 0 && ((claims.Power>>1)&1) == 1

		var count int64
		if db == nil {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}

		if err := db.Model(&user.Admin{}).Where("user_id = ?", claims.UserID).Count(&count).Error; err != nil {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}
		isAdminUser := count > 0

		if !isAdminToken || !isAdminUser {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
