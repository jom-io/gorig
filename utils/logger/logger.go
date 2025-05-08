package logger

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/consts"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/sys"
	"github.com/rs/xid"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"reflect"
	"strings"
)

type Level = string

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
)

func LevelOf(level Level) zapcore.Level {
	zapLevel := zapcore.InfoLevel
	switch level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.ErrorLevel
	}
	return zapLevel
}

func GetLogger(key string) *zap.Logger {
	rootPath := configure.GetString("logger."+key+".root", "./.logs/"+key+"/")
	logLevel := configure.GetString("logger."+key+".level", "debug")
	fileName := rootPath + configure.GetString("logger."+key+".file", key+".jsonl")
	maxSize := configure.GetInt("logger."+key+".size.max", 128)
	maxBackups := configure.GetInt("logger."+key+".backup.max", 30)
	maxAge := configure.GetInt("logger."+key+".age.max", 7)
	compress := configure.GetBool("logger."+key+".compress", false)
	sys.Warn("# Initialize the " + key + " log system ..... #")
	sys.Info(" * PATH: ", strings.ToUpper(rootPath), "      ${ logger."+key+".root }")
	sys.Info(" * LEVEL: ", strings.ToUpper(logLevel), "      ${ logger."+key+".level }")
	sys.Info(" * FILE: ", strings.ToUpper(fileName), "      ${ logger."+key+".file }")
	sys.Info(" * MAX SIZE: ", maxSize, "      ${ logger."+key+".size.max }")
	sys.Info(" * BACKUP MAX: ", maxBackups, "      ${ logger."+key+".backup.max }")
	sys.Info(" * AGE MAX: ", maxAge, "      ${ logger."+key+".age.max }")
	sys.Info(" * COMPRESS: ", compress, "      ${ logger."+key+".compress }")

	zapLogLevel := LevelOf(logLevel)

	hook := lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        key,
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zapLogLevel)

	writeSyncers := []zapcore.WriteSyncer{
		zapcore.AddSync(&hook),
	}

	if sys.RunMode.IsRd() {
		writeSyncers = append(writeSyncers, zapcore.AddSync(os.Stdout))
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writeSyncers...),
		atomicLevel,
	)

	sys.Success("# Initialize the " + key + " log system [OK] #")

	return zap.New(core)
}

func isNilPointer(i any) bool {
	if i == nil {
		return true
	}
	val := reflect.ValueOf(i)
	return val.Kind() == reflect.Ptr && val.IsNil()
}

func GetTraceID(ctx context.Context) string {
	return cast.ToString(getTraceID(ctx))
}

func getTraceID(ctx context.Context) any {
	if ctx == nil {
		return "no traceid"
	}
	if ginCtx, ok := ctx.(*gin.Context); ok && ginCtx != nil {
		return ginCtx.Value(consts.TraceIDKey)
	}
	if isNilPointer(ctx) {
		return "no-traceid"
	}
	return ctx.Value(consts.TraceIDKey)
}

func getUserID(ctx context.Context) any {
	if ctx == nil {
		return nil
	}
	if ginCtx, ok := ctx.(*gin.Context); ok && ginCtx != nil {
		return ginCtx.Value(consts.UserIDKey)
	}
	if isNilPointer(ctx) {
		return nil
	}
	return ctx.Value(consts.UserIDKey)
}

func putTraceId(ctx context.Context, fields ...zap.Field) []zap.Field {
	defer func() {
		if r := recover(); r != nil {
			sys.Warn("putTraceId panic:", r)
		}
	}()
	if ctx != nil {
		if userID := getUserID(ctx); userID != nil {
			fields = append([]zap.Field{zap.Any(consts.UserIDKey, userID)}, fields...)
		}
	}
	return append([]zap.Field{zap.Any(consts.TraceIDKey, getTraceID(ctx))}, fields...)
}

func NewCtx(traceIds ...string) context.Context {
	ctx := context.Background()
	var traceId string
	if len(traceIds) > 0 {
		traceId = traceIds[0]
	} else {
		traceId = xid.New().String()
	}
	return context.WithValue(ctx, consts.TraceIDKey, traceId)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.Info(msg, putTraceId(ctx, fields...)...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.Warn(msg, putTraceId(ctx, fields...)...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.Error(msg, putTraceId(ctx, fields...)...)
}

func DPanic(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.DPanic(msg, putTraceId(ctx, fields...)...)
}

func Panic(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.Panic(msg, putTraceId(ctx, fields...)...)
}

func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.Fatal(msg, putTraceId(ctx, fields...)...)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = insertLine(fields...)
	Logger.Debug(msg, putTraceId(ctx, fields...)...)
}

func insertLine(fields ...zap.Field) []zap.Field {
	//pc, file, line, ok := runtime.Caller(2)
	//if ok {
	//	// 获取函数名
	//	funcName := runtime.FuncForPC(pc).Name()
	//	fields = append(fields, zap.String("line", file+":"+funcName+":"+cast.ToString(line)))
	//}
	return fields
}

var Logger *zap.Logger
var Console *zap.Logger

func init() {
	Logger = GetLogger("commons")
	Console = GetLogger("console")
	if !sys.RunMode.IsRd() {
		sys.ConsoleToLogger(func(msg string) {
			Console.Info(msg)
		})
	}
}
