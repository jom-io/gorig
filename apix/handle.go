package apix

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix/response"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/notify/dingding"
	"runtime/debug"
)

func HandlePanic(ctx *gin.Context) {
	if r := recover(); r != nil {
		PanicNotify(ctx, r)
		response.ErrorSystem(ctx, GetTraceID(ctx), GetTraceID(ctx))
	}
}

func PanicNotify(ctx *gin.Context, err interface{}) {
	if err == nil {
		return
	}
	response.ErrorSystem(ctx, GetTraceID(ctx), GetTraceID(ctx))
	debug.PrintStack()
	log := fmt.Sprintf("TraceID: %s,\nPanic: %v, \nRequest: %v,  \nStack: %s", GetTraceID(ctx), err, ctx.Request, string(debug.Stack()))
	logger.DPanic(ctx, log)
	go dingding.PanicNotifyDefault(log)
}

func HandleError(ctx *gin.Context, code int, data *interface{}, error *errors.Error) {
	if error == nil {
		return
	}
	if error.Type == errors.System {
		logger.Warn(ctx, error.Error())
		response.ErrorSystem(ctx, GetTraceID(ctx), GetTraceID(ctx))
		log := fmt.Sprintf("TraceID: %s, \nError: %v, \nRequest: %v,  \nStack: %s", GetTraceID(ctx), error, ctx.Request, string(debug.Stack()))
		go dingding.ErrNotifyDefault(log)
	}
	if error.Type == errors.Application {
		logger.Error(ctx, error.Error())
		response.Fail(ctx, code, error.Message, data)
	}
}

func Handle(ctx *gin.Context, code int, err *errors.Error) {
	if err != nil {
		HandleError(ctx, code, nil, err)
	} else {
		response.S(ctx)
	}
}

func HandleData(ctx *gin.Context, code int, data interface{}, err *errors.Error) {
	if err != nil {
		if err.Code != "" && err.CodeInt() != 0 {
			code = err.CodeInt()
		}
		HandleError(ctx, code, &data, err)
	} else {
		response.Success(ctx, "", &data)
	}
}

func SendError(ctx *gin.Context, code int, message string, error *errors.Error) {
	if error == nil {
		error = errors.Verify(message)
	}
	log := fmt.Sprintf("TraceID: %s, \nError: %v, \nRequest: %v,  \nStack: %s", GetTraceID(ctx), error, ctx.Request, string(debug.Stack()))
	logger.Error(ctx, error.Error())
	go dingding.ErrNotifyDefault(log)
}
