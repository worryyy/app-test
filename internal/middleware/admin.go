package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

var adminForbiddenResp = responses.New(false, http.StatusForbidden, "权限不足")

func AdminPermissionCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetAdminClaims(c)
		if claims == nil {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}
		if db == nil {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}

		var admin user.Admin
		err := db.WithContext(c.Request.Context()).
			Where("id = ? AND user_id = ?", claims.AdminID, claims.UserID).
			Take(&admin).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}
		if err != nil {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}
		if admin.Power != claims.Power {
			adminForbiddenResp.Resp(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
