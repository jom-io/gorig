package om

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/variable"
	"github.com/jom-io/gorig/httpx"
	"github.com/jom-io/gorig/om/delploy/app"
	dpGit "github.com/jom-io/gorig/om/delploy/git"
	dpTask "github.com/jom-io/gorig/om/delploy/task"
	"github.com/jom-io/gorig/om/logtool"
	"github.com/jom-io/gorig/om/mid"
	"github.com/jom-io/gorig/om/omuser"
)

func init() {
	if variable.OMKey == "" {
		return
	}
	httpx.RegisterRouter(func(groupRouter *gin.RouterGroup) {
		om := groupRouter.Group("om")
		auth := om.Group("auth")
		auth.POST("connect", omuser.Login)
		om.Use(mid.Sign())
		log := om.Group("log")
		log.GET("categories", logtool.GetCategories)
		log.GET("levels", logtool.GetLevels)
		log.POST("search", logtool.Search)
		log.GET("near", logtool.Near)
		log.GET("monitor", logtool.Monitor)
		log.GET("download", logtool.Download)

		git := om.Group("git")
		git.GET("check", dpGit.CheckGit)
		git.GET("ssh/key", dpGit.GetSSHKey)
		git.POST("repo/set", dpGit.SetRepo)
		git.GET("branches/list", dpGit.ListBranches)
		git.POST("branch/associate", dpGit.AssociateBranch)
		//git.POST("auto", auto)

		deploy := om.Group("deploy")
		deploy.POST("restart", app.Restart)
		deploy.POST("stop", app.Stop)

		task := deploy.Group("task")
		task.POST("save", dpTask.SaveTask)

		//env := om.Group("env")
		//env.GET("cpu", host.Cpu)
		//env.GET("disk", disk)
		//env.GET("mem", mem)
	})
}
