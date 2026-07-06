package cronx

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/notify/dingding"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

const (
	persistScheduledKey  = "gorig:cronx:persist:scheduled"
	persistProcessingKey = "gorig:cronx:persist:processing"
	persistTaskKeyPrefix = "gorig:cronx:persist:task:"

	persistStatusPending = "pending"
	persistStatusRunning = "running"
	persistStatusDone    = "done"
	persistStatusFailed  = "failed"

	persistPollInterval = 500 * time.Millisecond
	persistBatchSize    = 100
	persistLease        = 30 * time.Minute
	persistDoneTTL      = 24 * time.Hour
)

// PersistPayload marks values that can be stored as persistent task payloads.
// Implementations must be JSON serializable and should keep json tags stable.
type PersistPayload interface {
	PersistPayload()
}

// PersistHandler is the fixed persistent task handler shape.
// A handler receives exactly one JSON-serializable payload.
type PersistHandler[P PersistPayload] func(ctx context.Context, payload P) error

type persistTask struct {
	ID            string          `json:"id"`
	Handler       string          `json:"handler"`
	Payload       json.RawMessage `json:"payload"`
	RunAt         int64           `json:"run_at"`
	Status        string          `json:"status"`
	TimeoutMillis int64           `json:"timeout_millis,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

type persistRegisteredHandler struct {
	name        string
	payloadType reflect.Type
	run         func(context.Context, json.RawMessage) error
}

var (
	persistRegistryMu sync.RWMutex
	persistRegistry   = map[string]persistRegisteredHandler{}

	persistWorkerMu     sync.Mutex
	persistWorkerCancel context.CancelFunc
	persistWorkerActive bool
	persistWorkerSeq    int64
)

var claimPersistDueScript = redis.NewScript(`
local ids = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, ARGV[2])
for _, id in ipairs(ids) do
	redis.call('ZREM', KEYS[1], id)
	redis.call('ZADD', KEYS[2], ARGV[3], id)
end
return ids
`)

var recoverPersistExpiredScript = redis.NewScript(`
local ids = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, ARGV[2])
for _, id in ipairs(ids) do
	redis.call('ZREM', KEYS[1], id)
	redis.call('ZADD', KEYS[2], ARGV[1], id)
end
return ids
`)

// RegisterPersistTask registers a persistent task handler at application startup.
//
// Use it for every handler that may be referenced by AddPersistDelayTask or
// AddPersistOnceTask. Redis only stores the handler name and JSON payload; after
// a process restart, cronx needs this startup registration to map pending Redis
// tasks back to the Go function.
func RegisterPersistTask[P PersistPayload](handler PersistHandler[P]) error {
	return registerPersistTask(handler)
}

// AddPersistDelayTask adds a persistent one-shot task that runs after delay.
// The payload is JSON serialized into Redis and restored when the task runs.
func AddPersistDelayTask[P PersistPayload](
	delay time.Duration,
	handler PersistHandler[P],
	payload P,
	timeout ...time.Duration,
) (string, error) {
	if delay <= 0 {
		return "", fmt.Errorf("invalid delay for persistent cron task: %s", delay)
	}
	return AddPersistOnceTask(time.Now().Add(delay), handler, payload, timeout...)
}

// AddPersistOnceTask adds a persistent one-shot task that runs at runAt.
func AddPersistOnceTask[P PersistPayload](
	runAt time.Time,
	handler PersistHandler[P],
	payload P,
	timeout ...time.Duration,
) (string, error) {
	name, err := persistHandlerName(handler)
	if err != nil {
		return "", err
	}
	if err := ensurePersistTaskRegistered[P](name); err != nil {
		return "", err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal persistent task payload: %w", err)
	}

	now := time.Now()
	task := persistTask{
		ID:        xid.New().String(),
		Handler:   name,
		Payload:   body,
		RunAt:     runAt.UnixMilli(),
		Status:    persistStatusPending,
		CreatedAt: now.UnixMilli(),
		UpdatedAt: now.UnixMilli(),
	}
	if len(timeout) > 0 && timeout[0] > 0 {
		task.TimeoutMillis = timeout[0].Milliseconds()
	}

	client, err := persistRedisClient(context.Background())
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("marshal persistent task: %w", err)
	}

	ctx := context.Background()
	if err := client.Set(ctx, persistTaskKey(task.ID), raw, 0).Err(); err != nil {
		return "", err
	}
	if err := client.ZAdd(ctx, persistScheduledKey, &redis.Z{
		Score:  float64(task.RunAt),
		Member: task.ID,
	}).Err(); err != nil {
		_ = client.Del(ctx, persistTaskKey(task.ID)).Err()
		return "", err
	}

	startPersistWorker()
	return task.ID, nil
}

func registerPersistTask[P PersistPayload](handler PersistHandler[P]) error {
	name, err := persistHandlerName(handler)
	if err != nil {
		return err
	}
	payloadType := reflect.TypeOf((*P)(nil)).Elem()

	persistRegistryMu.Lock()
	defer persistRegistryMu.Unlock()

	if current, ok := persistRegistry[name]; ok {
		if current.payloadType != payloadType {
			return fmt.Errorf("persistent task handler %s already registered with payload %s", name, current.payloadType)
		}
		return nil
	}

	persistRegistry[name] = persistRegisteredHandler{
		name:        name,
		payloadType: payloadType,
		run: func(ctx context.Context, raw json.RawMessage) error {
			var payload P
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &payload); err != nil {
					return fmt.Errorf("unmarshal persistent task payload for %s: %w", name, err)
				}
			}
			return handler(ctx, payload)
		},
	}
	return nil
}

func ensurePersistTaskRegistered[P PersistPayload](name string) error {
	payloadType := reflect.TypeOf((*P)(nil)).Elem()

	persistRegistryMu.RLock()
	defer persistRegistryMu.RUnlock()

	current, ok := persistRegistry[name]
	if !ok {
		return fmt.Errorf("persistent task handler %s is not registered; call cronx.RegisterPersistTask at startup", name)
	}
	if current.payloadType != payloadType {
		return fmt.Errorf("persistent task handler %s registered with payload %s, got %s",
			name, current.payloadType, payloadType)
	}
	return nil
}

func persistHandlerName(handler any) (string, error) {
	value := reflect.ValueOf(handler)
	if !value.IsValid() || value.Kind() != reflect.Func || value.IsNil() {
		return "", fmt.Errorf("persistent task handler must be a function")
	}
	fn := runtime.FuncForPC(value.Pointer())
	if fn == nil || fn.Name() == "" {
		return "", fmt.Errorf("persistent task handler name not found")
	}
	name := fn.Name()
	if strings.Contains(name, ".func") {
		return "", fmt.Errorf("persistent task handler must be a named function: %s", name)
	}
	return name, nil
}

func persistRedisClient(ctx context.Context) (*redis.Client, error) {
	redisCache := cache.GetRedisInstance[any](ctx)
	if redisCache == nil || !redisCache.IsInitialized() {
		return nil, fmt.Errorf("redis client is nil")
	}
	return redisCache.Client, nil
}

func startPersistWorker() {
	persistWorkerMu.Lock()
	defer persistWorkerMu.Unlock()

	if persistWorkerActive {
		return
	}
	client, err := persistRedisClient(context.Background())
	if err != nil {
		logger.Warn(nil, "persistent cron worker not started", zap.Error(err))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	persistWorkerSeq++
	seq := persistWorkerSeq
	persistWorkerCancel = cancel
	persistWorkerActive = true

	go func() {
		defer func() {
			persistWorkerMu.Lock()
			if persistWorkerSeq == seq {
				persistWorkerActive = false
				persistWorkerCancel = nil
			}
			persistWorkerMu.Unlock()
		}()
		runPersistWorker(ctx, client)
	}()
}

func stopPersistWorker() {
	persistWorkerMu.Lock()
	defer persistWorkerMu.Unlock()

	if persistWorkerCancel != nil {
		persistWorkerCancel()
	}
	persistWorkerSeq++
	persistWorkerActive = false
	persistWorkerCancel = nil
}

func runPersistWorker(ctx context.Context, client *redis.Client) {
	ticker := time.NewTicker(persistPollInterval)
	defer ticker.Stop()

	for {
		if err := pollPersistTasks(ctx, client); err != nil && ctx.Err() == nil {
			logger.Error(ctx, "poll persistent cron tasks failed", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func pollPersistTasks(ctx context.Context, client *redis.Client) error {
	now := time.Now().UnixMilli()
	if err := recoverPersistExpired(ctx, client, now); err != nil {
		return err
	}

	ids, err := claimPersistDue(ctx, client, now)
	if err != nil {
		return err
	}
	for _, id := range ids {
		go executePersistTask(ctx, client, id)
	}
	return nil
}

func recoverPersistExpired(ctx context.Context, client *redis.Client, now int64) error {
	_, err := recoverPersistExpiredScript.Run(ctx, client,
		[]string{persistProcessingKey, persistScheduledKey},
		now,
		persistBatchSize,
	).StringSlice()
	return err
}

func claimPersistDue(ctx context.Context, client *redis.Client, now int64) ([]string, error) {
	return claimPersistDueScript.Run(ctx, client,
		[]string{persistScheduledKey, persistProcessingKey},
		now,
		persistBatchSize,
		now+persistLease.Milliseconds(),
	).StringSlice()
}

func executePersistTask(ctx context.Context, client *redis.Client, id string) {
	task, err := loadPersistTask(ctx, client, id)
	if err != nil {
		logger.Error(ctx, "load persistent cron task failed", zap.String("task_id", id), zap.Error(err))
		_ = client.ZRem(ctx, persistProcessingKey, id).Err()
		return
	}
	if task.Status == persistStatusDone {
		_ = client.ZRem(ctx, persistProcessingKey, id).Err()
		return
	}

	persistRegistryMu.RLock()
	handler, ok := persistRegistry[task.Handler]
	persistRegistryMu.RUnlock()
	if !ok {
		task.LastError = "persistent task handler not registered"
		markPersistTaskFailed(ctx, client, task)
		return
	}

	task.Status = persistStatusRunning
	task.UpdatedAt = time.Now().UnixMilli()
	if err := savePersistTask(ctx, client, task, 0); err != nil {
		logger.Error(ctx, "save persistent cron running status failed", zap.String("task_id", task.ID), zap.Error(err))
	}

	if err := runPersistHandler(ctx, handler, task); err != nil {
		task.LastError = err.Error()
		markPersistTaskFailed(ctx, client, task)
		return
	}

	task.Status = persistStatusDone
	task.LastError = ""
	task.UpdatedAt = time.Now().UnixMilli()
	if err := savePersistTask(ctx, client, task, persistDoneTTL); err != nil {
		logger.Error(ctx, "save persistent cron done status failed", zap.String("task_id", task.ID), zap.Error(err))
	}
	if err := client.ZRem(ctx, persistProcessingKey, task.ID).Err(); err != nil {
		logger.Error(ctx, "remove persistent cron processing task failed", zap.String("task_id", task.ID), zap.Error(err))
	}
}

func runPersistHandler(ctx context.Context, handler persistRegisteredHandler, task *persistTask) error {
	call := func(runCtx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log := fmt.Sprintf("TraceID: %s,\nfunc: %v,\nPanic: %v,\nStack: %s",
					logger.GetTraceID(runCtx), handler.name, r, string(debug.Stack()))
				logger.DPanic(runCtx, "persistent cron task panic recovered",
					zap.String("func", handler.name),
					zap.String("task_id", task.ID),
					zap.Any("recover", r),
					zap.String("stack", string(debug.Stack())),
				)
				go dingding.PanicNotifyDefault(log)
				err = fmt.Errorf("persistent cron task panic: %v", r)
			}
		}()
		return handler.run(runCtx, task.Payload)
	}

	runCtx := logger.NewCtx()
	if task.TimeoutMillis <= 0 {
		return call(runCtx)
	}

	timeout := time.Duration(task.TimeoutMillis) * time.Millisecond
	runCtx, cancel := context.WithTimeout(runCtx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- call(runCtx)
	}()

	select {
	case err := <-done:
		return err
	case <-runCtx.Done():
		return runCtx.Err()
	}
}

func loadPersistTask(ctx context.Context, client *redis.Client, id string) (*persistTask, error) {
	raw, err := client.Get(ctx, persistTaskKey(id)).Bytes()
	if err != nil {
		return nil, err
	}
	task := &persistTask{}
	if err := json.Unmarshal(raw, task); err != nil {
		return nil, err
	}
	return task, nil
}

func savePersistTask(ctx context.Context, client *redis.Client, task *persistTask, ttl time.Duration) error {
	raw, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return client.Set(ctx, persistTaskKey(task.ID), raw, ttl).Err()
}

func markPersistTaskFailed(ctx context.Context, client *redis.Client, task *persistTask) {
	task.Status = persistStatusFailed
	task.UpdatedAt = time.Now().UnixMilli()
	if err := savePersistTask(ctx, client, task, persistDoneTTL); err != nil {
		logger.Error(ctx, "save persistent cron failed status failed", zap.String("task_id", task.ID), zap.Error(err))
	}
	if err := client.ZRem(ctx, persistProcessingKey, task.ID).Err(); err != nil {
		logger.Error(ctx, "remove failed persistent cron processing task failed", zap.String("task_id", task.ID), zap.Error(err))
	}
}

func persistTaskKey(id string) string {
	return persistTaskKeyPrefix + id
}
