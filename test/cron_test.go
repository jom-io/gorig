package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jom-io/gorig/cronx"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestAddCronTask_BasicExecution(t *testing.T) {
	var count int32
	cronx.AddCronTask("@every 1s", func(ctx context.Context) {
		atomic.AddInt32(&count, 1)
	}, 2*time.Second)

	// Start cron
	go cronx.Startup("CRON", "")

	// Wait for execution
	time.Sleep(2500 * time.Millisecond)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&count), int32(2), "Task should be executed at least twice")

	_ = cronx.Shutdown("CRON", context.Background())
}

func TestAddCronTask_WithPanic(t *testing.T) {
	cronx.AddCronTask("@every 1s", func(ctx context.Context) {
		panic("simulated panic in task")
	})

	go cronx.Startup("CRON", "")
	time.Sleep(2 * time.Second)
	_ = cronx.Shutdown("CRON", context.Background())
	// Manually check logs for recovered panic message
}

func TestAddCronTask_WithTimeout(t *testing.T) {
	cronx.AddCronTask("@every 1s", func(ctx context.Context) {
		select {
		case <-time.After(3 * time.Second):
			t.Log("should not reach here")
		case <-ctx.Done():
			// Check logs for timeout warning
		}
	}, 1*time.Second)

	go cronx.Startup("CRON", "")
	time.Sleep(2500 * time.Millisecond)
	_ = cronx.Shutdown("CRON", context.Background())
}

func TestAddTask_Deprecated(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Deprecated AddTask should not panic")
		}
	}()

	cronx.AddTask("@every 1s", func() {
		_ = errors.New("legacy task logic")
	})
	go cronx.Startup("CRON", "")
	time.Sleep(2 * time.Second)
	_ = cronx.Shutdown("CRON", context.Background())
}

func TestAddCronTask_LongRunningTask(t *testing.T) {
	cronx.AddCronTask("@every 10s", func(ctx context.Context) {
		t.Log(fmt.Sprintf("Long running task started at %s", time.Now().Format(time.RFC3339)))
		time.Sleep(15 * time.Second)
	})

	go cronx.Startup("CRON", "")
	time.Sleep(30 * time.Second) // 等待任务执行
	_ = cronx.Shutdown("CRON", context.Background())
}

func TestAddEveryTask_BasicExecution(t *testing.T) {
	interval := 1 * time.Second
	fmt.Println(fmt.Sprintf("@every %s", interval))
	var count int32
	cronx.AddEveryTask(interval, func(ctx context.Context) {
		atomic.AddInt32(&count, 1)
		t.Log(fmt.Sprintf("Task executed at %s, count: %d", time.Now().Format(time.RFC3339), atomic.LoadInt32(&count)))
	}, 2*time.Second)

	// Start cron
	go cronx.Startup("CRON", "")

	// Wait for execution
	time.Sleep(2500 * time.Millisecond)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&count), int32(2), "Task should be executed at least twice")

	_ = cronx.Shutdown("CRON", context.Background())
}

func TestAddDelayTask_BasicExecution(t *testing.T) {
	var count int32
	delay := 1 * time.Second
	cronx.AddDelayTask(delay, func(ctx context.Context) {
		atomic.AddInt32(&count, 1)
		t.Logf("Delayed task executed at %s, count: %d",
			time.Now().Format(time.RFC3339), atomic.LoadInt32(&count))
	}, 10*time.Second)

	time.Sleep(3 * time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&count), "Delayed task should be executed once")

	_ = cronx.Shutdown("CRON", context.Background())
}
