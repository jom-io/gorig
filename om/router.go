package om

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/httpx"
)

func init() {
	httpx.RegisterRouter(func(groupRouter *gin.RouterGroup) {
		om := groupRouter.Group("om")
		log := om.Group("log")
		log.GET("allCat", AllCat)
		log.GET("allLevel", AllLevel)
		log.POST("search", Search)
		log.GET("near", Near)
		log.GET("monitor", Monitor)
		log.GET("download", Download)
	})
}
