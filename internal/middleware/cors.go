package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		origin := c.GetHeader("Origin")
		if origin != "" {
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Vary", "Origin")
		}
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		header.Set("Access-Control-Allow-Headers",
			"Content-Type, Authorization, X-Requested-With, Accept, Origin, Referer, User-Agent")
		header.Set("Access-Control-Expose-Headers",
			"Content-Type, Content-Length, Authorization")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Max-Age", "3600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
