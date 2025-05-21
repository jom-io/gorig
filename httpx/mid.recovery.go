package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		logger.Error(c, "gin recovery", zap.Any("err", err))
		apix.PanicNotify(c, err)
	})
}
