package serv

import (
	"context"
	"fmt"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strings"
	"time"
)

type Service struct {
	Code     string
	PORT     string
	Startup  func(code, port string) error
	Shutdown func(code string, ctx context.Context) error
}

var gServices map[string]Service

func doRegisterService(service Service) *errors.Error {
	_, exists := gServices[service.Code]
	if exists {
		return errors.Sys(fmt.Sprintf("The same service has been register.[ code=%s ]", service.Code))
	}
	gServices[service.Code] = service
	return nil
}

func RegisterService(service ...Service) *errors.Error {
	if len(service) == 0 {
		return errors.Sys("no any service")
	}
	for _, s := range service {
		err := doRegisterService(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func Running() {
	for code, service := range gServices {
		sys.Warn("# Start the service: ", code, " ...... #")
		err := service.Startup(code, service.PORT)
		if err != nil {
			logger.Logger.Error("start server failed", zap.String("code", code), zap.Error(err))
			sys.Error("# Start service exception: ", code, " #")
			return
		}
		sys.Success("# Start the service ", code, " [OK] #")
	}

	sys.Info("# ALL Used Configure Items #")
	configure.Dump(func(key string, val any) {
		if strings.Index(strings.ToLower(key), "pass") > -1 || strings.Index(strings.ToLower(key), "secret") > -1 || strings.Index(strings.ToLower(key), "key") > -1 {
			sys.Info("  # ", key, " # ==>> ", "**********")
		} else {
			sys.Info("  # ", key, " # ==>> ", val)
		}

	})

	sys.Success("# System startup successful #")

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

	sys.Info("# Shutting down the system ...... #")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for code, service := range gServices {
		sys.Info(" * Start stop service: ", code, " ......")
		err := service.Shutdown(code, ctx)
		if err != nil {
			logger.Logger.Error("shutdown server failed", zap.String("code", code), zap.Error(err))
			sys.Error(" * Stop service ", code, " exception")
		}
		sys.Success(" * Stop service ", code, " [OK]")
	}
	sys.Success("# Shutting down the system [OK] #")
}

func init() {
	gServices = make(map[string]Service)
}
