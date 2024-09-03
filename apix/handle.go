package apix

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix/response"
	"github.com/jom-io/gorig/httpx"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/notify/dingding"
	"runtime/debug"
)

func HandlePanic(ctx *gin.Context) {
	if r := recover(); r != nil {
		response.ErrorSystem(ctx, httpx.GetTraceID(ctx), httpx.GetTraceID(ctx))
		debug.PrintStack()
		log := fmt.Sprintf("TraceID: %s,\nPanic: %v, \nRequest: %v,  \nStack: %s", httpx.GetTraceID(ctx), r, ctx.Request, string(debug.Stack()))
		logger.DPanic(ctx, log)
		go dingding.PanicNotifyDefault(log)
	}
}

func HandleError(ctx *gin.Context, code int, data *interface{}, error *errors.Error) {
	if error == nil {
		return
	}
	logger.Error(ctx, error.Message)
	if error.Type == errors.System {
		response.ErrorSystem(ctx, httpx.GetTraceID(ctx), httpx.GetTraceID(ctx))
		log := fmt.Sprintf("TraceID: %s, \nError: %v, \nRequest: %v,  \nStack: %s", httpx.GetTraceID(ctx), error, ctx.Request, string(debug.Stack()))
		go dingding.ErrNotifyDefault(log)
	}
	if error.Type == errors.Application {
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
	log := fmt.Sprintf("TraceID: %s, \nError: %v, \nRequest: %v,  \nStack: %s", httpx.GetTraceID(ctx), error, ctx.Request, string(debug.Stack()))
	logger.Error(ctx, error.Error())
	go dingding.ErrNotifyDefault(log)
}
