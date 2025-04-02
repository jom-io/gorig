package httpx

import (
	"context"
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/jom-io/gorig/domainx"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/sys"
	"net/http"
	"time"
)

func IsRegistered() bool {
	return gHttpServer != nil
}

func Startup(code, port string) error {
	if gHttpServer != nil {
		sys.Info(" * Rest service already started")
		return nil
		//sys.Exit(errors.Sys("You should not start the rest service twice"))
	}
	gHttpServer = &http.Server{
		Addr:              port,
		Handler:           gEngine,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	sys.Info(" * Rest service startup on: ", gHttpServer.Addr)
	go func() {
		err := gHttpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			sys.Error(" * rest service listen failed")
			sys.Exit(errors.Sys(err.Error()))
			return
		}
	}()

	return nil
}

func Shutdown(code string, ctx context.Context) error {
	if err := gHttpServer.Shutdown(ctx); err != nil {
		sys.Error(" * Rest service shutdown error: ", err.Error())
		return err
	}

	sys.Error(" * Rest service exist: ", gHttpServer.Addr)
	return nil
}

// RegisterRouter 注册路由
func RegisterRouter(reg func(groupRouter *gin.RouterGroup)) {
	reg(&gEngine.RouterGroup)
}

func RegisterRouterMid(group func(groupRouter *gin.RouterGroup, mid ...gin.HandlerFunc) *gin.RouterGroup, mid ...gin.HandlerFunc) *gin.RouterGroup {
	var groupRouter *gin.RouterGroup
	RegisterRouter(func(groupRouter *gin.RouterGroup) {
		groupRouter = group(groupRouter, mid...)
	})
	return groupRouter
}

var gEngine = gin.New()
var gHttpServer *http.Server

func init() {
	gEngine.Use(CORS())
	gEngine.Use(Logger())
	gEngine.Use(Debounce(200 * time.Millisecond))
	//gEngine.Use(IdemVerify())
	//gEngine.Use(SignVerify())
	gEngine.Use(gzip.Gzip(gzip.DefaultCompression))
	if !sys.RunMode.IsRd() {
		gin.SetMode(gin.ReleaseMode)
	}
	RegisterRouter(func(groupRouter *gin.RouterGroup) {
		groupRouter.GET("ping", func(ctx *gin.Context) {
			Success(ctx, gin.H{
				"timestamp": fmt.Sprintf("%d", time.Now().UnixMilli()),
			})
		})
	})
}
