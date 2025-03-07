package om

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/om/logtool"
	"github.com/jom-io/gorig/om/user"
)

func login(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	pwd, e := apix.GetParamType[string](ctx, "pwd", apix.Force)
	if e != nil {
		return
	}
	result, err := user.Login(ctx, pwd)
	apix.HandleData(ctx, consts.CurdSelectFailCode, result, err)
}

func categories(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &logtool.Categories, nil)
}

func levels(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &logtool.Levels, nil)
}

func search(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	opts := logtool.SearchOptions{}
	e := apix.BindParams(ctx, &opts)
	if e != nil {
		return
	}
	result, err := logtool.SearchLogs(opts)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
}

func near(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	path, e := apix.GetParamType[string](ctx, "path", apix.Force)
	cenLine, e := apix.GetParamType[int64](ctx, "line", apix.Force)
	ctxRange, e := apix.GetParamType[int64](ctx, "range", apix.Force)
	if e != nil {
		return
	}
	result, err := logtool.FetchContextLines(path, cenLine, ctxRange)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
}

func monitor(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	opts := logtool.SearchOptions{}
	e := apix.BindParams(ctx, &opts)
	if e != nil {
		return
	}
	err := logtool.MonitorLogs(ctx, opts)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}

func download(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	path, e := apix.GetParamType[string](ctx, "path", apix.Force)
	if e != nil {
		return
	}
	err := logtool.DownloadLogs(ctx, path)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}
