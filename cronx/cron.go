package cronx

import (
	"context"
	"fmt"
	"github.com/jom-io/gorig/serv"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/notify/dingding"
	"github.com/jom-io/gorig/utils/sys"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

var c *cron.Cron

type Task struct {
	Spec string
	Name string
	Func func()
}

var (
	taskList = TaskSlice{}
	taskMux  sync.Mutex
)

type TaskSlice []*Task

func isTaskExists(spec, name string) bool {
	for _, t := range taskList {
		if t.Spec == spec && t.Name == name {
			return true
		}
	}
	return false
}

// Deprecated: Use AddCronTask instead. This version does not support context propagation or panic recovery
func AddTask(spec string, f func()) {
	taskMux.Lock()
	defer taskMux.Unlock()

	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if isTaskExists(spec, name) {
		logger.Warn(nil, "duplicate task ignored", zap.String("name", name), zap.String("spec", spec))
		return
	}

	taskList = append(taskList, &Task{Spec: spec, Name: name, Func: f})
}

func AddCronTask(spec string, f func(ctx context.Context), timeout ...time.Duration) {
	taskMux.Lock()
	defer taskMux.Unlock()

	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if isTaskExists(spec, name) {
		logger.Warn(nil, "duplicate task ignored", zap.String("name", name), zap.String("spec", spec))
		return
	}
	taskList = append(taskList, &Task{
		Spec: spec,
		Name: name,
		Func: WrapCronTask(f, timeout...),
	})
}

func WrapCronTask(f func(ctx context.Context), timeout ...time.Duration) func() {
	baseCtx := logger.NewCtx()
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()

	return func() {
		ctx := baseCtx
		var cancel context.CancelFunc
		if len(timeout) > 0 && timeout[0] > 0 {
			ctx, cancel = context.WithTimeout(baseCtx, timeout[0])
			defer cancel()
		}

		done := make(chan struct{})

		go func() {
			defer func() {
				if r := recover(); r != nil {
					debug.PrintStack()
					log := fmt.Sprintf("TraceID: %s,\nfunc: %v,\nPanic: %v,\nStack: %s",
						logger.GetTraceID(ctx), name, r, string(debug.Stack()))
					logger.DPanic(ctx, "cron job panic recovered",
						zap.String("func", name),
						zap.Any("recover", r),
						zap.String("stack", string(debug.Stack())),
					)
					go dingding.PanicNotifyDefault(log)
				}
				close(done)
			}()
			f(ctx)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			logger.Error(ctx, "cron job timeout or canceled",
				zap.String("func", name),
				zap.Duration("timeout", timeout[0]),
				zap.Error(ctx.Err()),
			)
		}
	}
}

func start() {
	c = cron.New(cron.WithSeconds())
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
