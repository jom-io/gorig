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

var ins = func() *cron.Cron {
	if c == nil {
		c = cron.New(cron.WithSeconds(), cron.WithChain(
			cron.Recover(&loggerAdapter{}),
		))
	}
	return c
}

type task struct {
	EntryID cron.EntryID
	Spec    string
	Name    string
	Func    func()
}

var (
	taskList = taskSlice{}
	taskMux  sync.Mutex
)

type taskSlice []*task

func isTaskExists(spec, name string) bool {
	for _, t := range taskList {
		if t.Spec == spec && t.Name == name {
			return true
		}
	}
	return false
}

func (t *task) NextTime() time.Time {
	if c == nil {
		return time.Time{}
	}
	entry := c.Entry(t.EntryID)
	return entry.Next
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

	taskList = append(taskList, &task{Spec: spec, Name: name, Func: f})
}

func AddCronTask(spec string, f func(ctx context.Context), timeout ...time.Duration) {
	taskMux.Lock()
	defer taskMux.Unlock()

	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if isTaskExists(spec, name) {
		logger.Warn(nil, "duplicate task ignored", zap.String("name", name), zap.String("spec", spec))
		return
	}
	taskList = append(taskList, &task{
		Spec: spec,
		Name: name,
		Func: WrapCronTask(f, nil, timeout...),
	})
}

// AddEveryTask is a convenience function to add a task that runs at a regular interval.
func AddEveryTask(interval time.Duration, f func(ctx context.Context), timeout ...time.Duration) {
	taskMux.Lock()
	defer taskMux.Unlock()

	if interval <= time.Millisecond {
		logger.Warn(nil, "invalid interval for cron task", zap.Duration("interval", interval))
		return
	}
	spec := fmt.Sprintf("@every %s", interval)
	AddCronTask(spec, f, timeout...)
}

type onceSchedule struct {
	runAt time.Time
	done  bool
}

func (o *onceSchedule) Next(t time.Time) time.Time {
	if o.done {
		return time.Time{}
	}
	if t.After(o.runAt.Add(100 * time.Millisecond)) {
		o.done = true
		return time.Time{}
	}
	if t.Before(o.runAt) {
		return o.runAt
	}
	o.done = true
	return t.Add(10 * time.Millisecond)
}

// AddDelayTask adds a task that runs after a specified delay.
func AddDelayTask(delay time.Duration, f func(ctx context.Context), timeout ...time.Duration) {
	if delay <= 0 {
		logger.Warn(nil, "invalid delay for cron task", zap.Duration("delay", delay))
		return
	}
	runAt := time.Now().Add(delay)
	AddOnceTask(runAt, f, timeout...)
}

// AddOnceTask adds a task that runs only once at the specified time.
func AddOnceTask(runAt time.Time, f func(ctx context.Context), timeout ...time.Duration) {
	taskMux.Lock()
	defer taskMux.Unlock()

	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if isTaskExists(runAt.Format(time.RFC3339), name) {
		logger.Warn(nil, "duplicate once task ignored",
			zap.String("name", name),
			zap.Time("runAt", runAt),
		)
		return
	}

	var entryID cron.EntryID
	schedule := &onceSchedule{runAt: runAt}
	job := cron.FuncJob(WrapCronTask(f, func() {
		ins().Remove(entryID)
	}, timeout...))

	entryID = ins().Schedule(schedule, job)

	c.Start()
}

func WrapCronTask(f func(ctx context.Context), doneCallback func(), timeout ...time.Duration) func() {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()

	return func() {
		ctx := logger.NewCtx()
		var cancel context.CancelFunc
		if len(timeout) > 0 && timeout[0] > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout[0])
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

		if doneCallback != nil {
			doneCallback()
		}
	}
}

func start() {
	if c == nil {
		c = ins()
	}
	for _, t := range taskList {
		sys.Info("  * Add cron job", zap.String("name", t.Name), zap.String("spec", t.Spec))
		id, err := c.AddFunc(t.Spec, t.Func)
		if err != nil {
			logger.Fatal(nil, "  * Add cron job failed", zap.Error(err))
		}
		t.EntryID = id
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
