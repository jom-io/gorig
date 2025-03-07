package om

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/httpx"
	"github.com/jom-io/gorig/om/mid"
)

func init() {
	httpx.RegisterRouter(func(groupRouter *gin.RouterGroup) {
		om := groupRouter.Group("om")
		auth := om.Group("auth")
		auth.POST("connect", login)
		om.Use(mid.Sign())
		log := om.Group("log")
		log.GET("categories", categories)
		log.GET("levels", levels)
		log.POST("search", search)
		log.GET("near", near)
		log.GET("monitor", monitor)
		log.GET("download", download)
	})
}
