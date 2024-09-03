package gormt

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	configure "gorig/utils/cofigure"
	"gorig/utils/logger"
	gormLog "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"strings"
	"time"
)

// 自定义日志格式, 对 gorm 自带日志进行拦截重写
func createCustomGormLog(sqlType string, options ...Options) gormLog.Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)
	logConf := gormLog.Config{
		SlowThreshold: time.Second * configure.GetDuration(sqlType+".SlowThreshold"),
		LogLevel:      gormLog.Warn,
		Colorful:      false,
	}
	log := &loggerc{
		Writer:       logOutPut{},
		Config:       logConf,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
	for _, val := range options {
		val.apply(log)
	}
	return log
}

type logOutPut struct{}

func (l logOutPut) Printf(strFormat string, args ...interface{}) {
	logRes := fmt.Sprintf(strFormat, args...)
	logFlag := "gorm_v2 日志:"
	detailFlag := "详情："
	if strings.HasPrefix(strFormat, "[info]") || strings.HasPrefix(strFormat, "[traceStr]") {
		logger.Logger.Info(logFlag, zap.String(detailFlag, logRes))
	} else if strings.HasPrefix(strFormat, "[error]") || strings.HasPrefix(strFormat, "[traceErr]") {
		logger.Logger.Error(logFlag, zap.String(detailFlag, logRes))
	} else if strings.HasPrefix(strFormat, "[warn]") || strings.HasPrefix(strFormat, "[traceWarn]") {
		logger.Logger.Warn(logFlag, zap.String(detailFlag, logRes))
	}

}

// Options 尝试从外部重写内部相关的格式化变量
type Options interface {
	apply(*loggerc)
}
type OptionFunc func(log *loggerc)

func (f OptionFunc) apply(log *loggerc) {
	f(log)
}

// SetInfoStrFormat 定义 6 个函数修改内部变量
func SetInfoStrFormat(format string) Options {
	return OptionFunc(func(log *loggerc) {
		log.infoStr = format
	})
}

func SetWarnStrFormat(format string) Options {
	return OptionFunc(func(log *loggerc) {
		log.warnStr = format
	})
}

func SetErrStrFormat(format string) Options {
	return OptionFunc(func(log *loggerc) {
		log.errStr = format
	})
}

func SetTraceStrFormat(format string) Options {
	return OptionFunc(func(log *loggerc) {
		log.traceStr = format
	})
}
func SetTracWarnStrFormat(format string) Options {
	return OptionFunc(func(log *loggerc) {
		log.traceWarnStr = format
	})
}

func SetTracErrStrFormat(format string) Options {
	return OptionFunc(func(log *loggerc) {
		log.traceErrStr = format
	})
}

type loggerc struct {
	gormLog.Writer
	gormLog.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// LogMode log mode
func (l *loggerc) LogMode(level gormLog.LogLevel) gormLog.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l loggerc) Info(_ context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLog.Info {
		l.Printf(l.infoStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Warn print warn messages
func (l loggerc) Warn(_ context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLog.Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Error print error messages
func (l loggerc) Error(_ context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLog.Error {
		l.Printf(l.errStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Trace print sql message
func (l loggerc) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormLog.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormLog.Error && (!errors.Is(err, gormLog.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-1", sql)
		} else {
			l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormLog.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-1", sql)
		} else {
			l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == gormLog.Info:
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-1", sql)
		} else {
			l.Printf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
