package test

import (
	"context"
	"testing"
	"time"

	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/cronx"
)

type persistDelayPayload struct {
	Value string `json:"value"`
}

func (persistDelayPayload) PersistPayload() {}

var persistDelayResult chan string

func handlePersistDelay(ctx context.Context, payload persistDelayPayload) error {
	persistDelayResult <- payload.Value
	return nil
}

type persistBadPayload struct {
	Ch chan int `json:"ch"`
}

func (persistBadPayload) PersistPayload() {}

func handlePersistBadPayload(ctx context.Context, payload persistBadPayload) error {
	return nil
}

type persistUnregisteredPayload struct {
	Value string `json:"value"`
}

func (persistUnregisteredPayload) PersistPayload() {}

func handlePersistUnregistered(ctx context.Context, payload persistUnregisteredPayload) error {
	return nil
}

func TestAddPersistDelayTask_ExecutesPayload(t *testing.T) {
	ctx := context.Background()
	redisCache := cache.GetRedisInstance[any](ctx)
	if redisCache == nil || !redisCache.IsInitialized() {
		t.Skip("redis not available")
	}

	cleanupPersistTaskKeys(t, redisCache)
	persistDelayResult = make(chan string, 1)

	if err := cronx.RegisterPersistTask(handlePersistDelay); err != nil {
		t.Fatalf("RegisterPersistTask failed: %v", err)
	}
	_, err := cronx.AddPersistDelayTask(
		100*time.Millisecond,
		handlePersistDelay,
		persistDelayPayload{Value: "ok"},
		time.Second,
	)
	if err != nil {
		t.Fatalf("AddPersistDelayTask failed: %v", err)
	}

	select {
	case value := <-persistDelayResult:
		if value != "ok" {
			t.Fatalf("unexpected payload value: %s", value)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("persistent delay task did not execute")
	}

	time.Sleep(200 * time.Millisecond)
	cleanupPersistTaskKeys(t, redisCache)
}

func TestAddPersistDelayTask_RejectsNonJSONPayload(t *testing.T) {
	if err := cronx.RegisterPersistTask(handlePersistBadPayload); err != nil {
		t.Fatalf("RegisterPersistTask failed: %v", err)
	}
	_, err := cronx.AddPersistDelayTask(
		time.Second,
		handlePersistBadPayload,
		persistBadPayload{Ch: make(chan int)},
	)
	if err == nil {
		t.Fatal("expected JSON marshal error")
	}
}

func TestAddPersistDelayTask_RequiresRegisteredHandler(t *testing.T) {
	_, err := cronx.AddPersistDelayTask(
		time.Second,
		handlePersistUnregistered,
		persistUnregisteredPayload{Value: "pending"},
	)
	if err == nil {
		t.Fatal("expected unregistered handler error")
	}
}

func cleanupPersistTaskKeys(t *testing.T, redisCache *cache.RedisCache[any]) {
	t.Helper()

	keys, err := redisCache.Client.Keys(context.Background(), "gorig:cronx:persist:*").Result()
	if err != nil {
		t.Fatalf("list persistent task keys failed: %v", err)
	}
	if len(keys) == 0 {
		return
	}
	if err := redisCache.Client.Del(context.Background(), keys...).Err(); err != nil {
		t.Fatalf("delete persistent task keys failed: %v", err)
	}
}
