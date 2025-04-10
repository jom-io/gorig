package cronx

import (
	"context"
	"github.com/jom-io/gorig/serv"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"reflect"
	"runtime"
)

var c *cron.Cron

type Task struct {
	Spec string
	Name string
	Func func()
}

var taskList = TaskSlice{}

type TaskSlice []*Task

func AddTask(spec string, f func()) {
	taskList = append(taskList, &Task{Spec: spec, Name: runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), Func: f})
}

func init() {
	c = cron.New(cron.WithSeconds())
}

func start() {
	if c != nil {
		c.Stop()
	}
	for _, t := range taskList {
		sys.Info("  * Add cron job", zap.String("name", t.Name), zap.String("spec", t.Spec))
		if _, err := c.AddFunc(t.Spec, t.Func); err != nil {
			logger.Fatal(nil, "  * Add cron job failed", zap.Error(err))
		}
	}
	if len(taskList) == 0 {
		return
	}
	c.Start()
}

func Startup(code, port string) error {
	sys.Info("  * Cron service startup")
	start()
	return nil
}

func Shutdown(code string, ctx context.Context) error {
	sys.Info("  * Cron service shutdown")
	if c != nil {
		c.Stop()
	}
	return nil
}

func init() {
	err := serv.RegisterService(
		serv.Service{
			Code:     "CRON",
			Startup:  Startup,
			Shutdown: Shutdown,
		},
	)
	if err != nil {
		sys.Exit(err)
	}
}
