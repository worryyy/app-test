package agentchat

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
)

func currentUserID(c *gin.Context) int64 {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return 0
	}
	return claims.UserID
}

func currentRootUserID(c *gin.Context) int64 {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
	}
	return claims.UserID
}

func currentAccountType(c *gin.Context) string {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return ""
	}
	return claims.AccountType
}
