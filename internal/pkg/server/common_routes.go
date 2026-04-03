package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterCommonRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
}