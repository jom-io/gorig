package bootstrap

import (
	"github.com/gin-gonic/gin"
	_ "gorig/cronx"
	_ "gorig/domainx"
	"gorig/httpx"
	"gorig/serv"
	"gorig/utils/sys"
)

func StartUp() {
	sys.Warn("# All registered API information ...... #")
	httpx.DumpRouters(func(info gin.RouteInfo) {
		sys.Info(" * ", info.Method, ": ", info.Path)
	})

	sys.Success("# All registered API information [OK] #")
	serv.Running()
}
