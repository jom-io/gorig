package delpoy

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
	"github.com/jom-io/gorig/global/consts"
)

func SaveTask(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	opts := &TaskOptions{}
	e := apix.BindParams(ctx, opts)
	if e != nil {
		return
	}
	err := Task.Save(ctx, opts)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}
