package bootstrap

import (
	"github.com/gin-gonic/gin"
	_ "github.com/jom-io/gorig/cache"
	_ "github.com/jom-io/gorig/cronx"
	_ "github.com/jom-io/gorig/domainx"
	_ "github.com/jom-io/gorig/global/variable"
	"github.com/jom-io/gorig/httpx"
	"github.com/jom-io/gorig/serv"
	"github.com/jom-io/gorig/utils/sys"
)

func StartUp() {
	sys.Warn("# All registered API information ...... #")
	httpx.DumpRouters(func(info gin.RouteInfo) {
		sys.Info(" * ", info.Method, ": ", info.Path)
	})

	sys.Success("# All registered API information [OK] #")
	serv.Running()
}
