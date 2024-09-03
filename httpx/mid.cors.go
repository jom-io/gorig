package httpx

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	AllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	AllowHeaders = "" +
		"Origin, " +
		"Content-Type, " +
		"Content-Length, " +
		"Accept-Encoding, " +
		"X-CSRF-Token, " +
		"Authorization, "
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", AllowMethods)
		c.Writer.Header().Set("Access-Control-Allow-Headers", AllowHeaders)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
