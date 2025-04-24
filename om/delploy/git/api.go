package delpoy

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
	"github.com/jom-io/gorig/global/consts"
)

func CheckGit(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	result, err := Git.CheckGit(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
}

func SetRepo(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	repo, e := apix.GetParamType[string](ctx, "repo", apix.Force)
	if e != nil {
		return
	}
	err := Git.SetRepo(ctx, repo)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}

func GetSSHKey(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	result, err := Git.GetSSHKey(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
}

func ListBranches(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	result, err := Git.ListBranches(ctx)
	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
}

func AssociateBranch(ctx *gin.Context) {
	defer apix.HandlePanic(ctx)
	branch, e := apix.GetParamType[string](ctx, "branch", apix.Force)
	if e != nil {
		return
	}
	err := Git.AssociateBranch(ctx, branch)
	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
}

//
//func setBranch(ctx *gin.Context) {
//	defer apix.HandlePanic(ctx)
//	branch, e := apix.GetParamType[string](ctx, "branch", apix.Force)
//	if e != nil {
//		return
//	}
//	err := logtool.SetBranch(branch)
//	apix.HandleData(ctx, consts.CurdSelectFailCode, nil, err)
//}
//
//func auto(ctx *gin.Context) {
//	defer apix.HandlePanic(ctx)
//	result, err := logtool.AutoCommit()
//	apix.HandleData(ctx, consts.CurdSelectFailCode, &result, err)
//}
