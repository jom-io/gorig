package om

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/om/logtool"
)

func AllCat(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &logtool.Categories, nil)
}

func AllLevel(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &logtool.Levels, nil)
}

func Search(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	opts := logtool.SearchOptions{}
	e := apix.BindParams(ctx, &opts)
	if e != nil {
		return
	}
	result, err := logtool.SearchLogs(opts)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
}

func Near(ctx *gin.Context) {
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

func Monitor(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	opts := logtool.SearchOptions{}
	e := apix.BindParams(ctx, &opts)
	if e != nil {
		return
	}
	err := logtool.MonitorLogs(ctx, opts)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}

func Download(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	path, e := apix.GetParamType[string](ctx, "path", apix.Force)
	if e != nil {
		return
	}
	err := logtool.DownloadLogs(ctx, path)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}
