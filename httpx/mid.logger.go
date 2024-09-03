package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
)

var restLogger = logger.GetLogger("rest")

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer restLogger.Sync()
		SetTraceID(c)
		restLogger.Info("IN", doGetArrForIn(c)...)

		c.Next()

		restLogger.Info("OUT", doGetArrForOut(c)...)
	}
}

func doGetArrForIn(c *gin.Context) []zap.Field {
	return []zap.Field{
		zap.String(consts.TraceIDKey, GetTraceID(c)),
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
		zap.String("remoteAddr", c.Request.RemoteAddr),
		zap.Any("header", c.Request.Header),
		zap.Any("query", c.Request.URL.Query()),
	}
}

func doGetArrForOut(c *gin.Context) []zap.Field {
	return []zap.Field{
		zap.String(consts.TraceIDKey, GetTraceID(c)),
		zap.Int("status", c.Writer.Status()),
	}
}
