package cronx

import (
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
)

type loggerAdapter struct {
}

func ts(keysAndValues ...interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok1 := keysAndValues[i].(string)
		val := keysAndValues[i+1]
		if ok1 {
			fields = append(fields, zap.Any(key, val))
		}
	}
	return fields
}

func (z *loggerAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	fields := ts(keysAndValues...)
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	logger.Error(nil, msg, fields...)
}

func (z *loggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	fields := ts(keysAndValues...)
	logger.Info(nil, msg, fields...)
}
