package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/sys"
	"github.com/spf13/cast"
	"sync"
	"time"
)

var (
	redisInstance *redis.Client
	initMu        sync.Mutex
	initOnce      = &sync.Once{}
)

func GetRedisInstance[T any]() *RedisCache[T] {
	initMu.Lock()
	defer initMu.Unlock()
	if redisInstance == nil {
		redisInstance = initRedisCache()
	}
	if redisInstance == nil {
		return nil
	}
	return &RedisCache[T]{
		Client: redisInstance,
		Ctx:    context.Background(),
	}
}

func initRedisCache() *redis.Client {
	initOnce.Do(func() {
		addr := configure.GetString("redis.addr")
		password := configure.GetString("redis.password")
		db := configure.GetString("redis.db")
		if addr == "" {
			sys.Info("# Redis addr is empty, skipping initialization")
		}

		cache, err := newRedisCache(RedisConfig{
			Addr:     addr,
			Password: password,
			DB:       cast.ToInt(db),
		})
		if err != nil {
			sys.Error("# failed to init Redis cache: ", err)
		}
		if cache == nil {
			sys.Error("# Redis cache is nil after initialization")
		}
		sys.Info("# Redis cache initialized")
		redisInstance = cache
	})
	return redisInstance
}

// RedisConfig holds the Redis configuration parameters
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RedisCache[T any] struct {
	Client *redis.Client
	Ctx    context.Context
}

func newRedisCache(cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return client, nil
}

func (r *RedisCache[T]) IsInitialized() bool {
	return r != nil && r.Client != nil
}

func (r *RedisCache[T]) Get(key string) (T, error) {
	var zero T
	if !r.IsInitialized() {
		return zero, fmt.Errorf("redis client is nil")
	}
	val, err := r.Client.Get(r.Ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return zero, ErrCacheMiss
	} else if err != nil {
		return zero, err
	}
	var data T
	if err = json.Unmarshal([]byte(val), &data); err != nil {
		return zero, err
	}
	return data, nil
}

func (r *RedisCache[T]) Set(key string, value T, expiration time.Duration) error {
	if !r.IsInitialized() {
		return fmt.Errorf("redis client is nil")
	}
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(r.Ctx, key, jsonValue, expiration).Err()
}

func (r *RedisCache[T]) Del(key string) error {
	if !r.IsInitialized() {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Del(r.Ctx, key).Err()
}

func (r *RedisCache[T]) Exists(key string) (bool, error) {
	if !r.IsInitialized() {
		return false, fmt.Errorf("redis client is nil")
	}
	result, err := r.Client.Exists(r.Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (r *RedisCache[T]) RPush(queue string, value T) error {
	if !r.IsInitialized() {
		return fmt.Errorf("redis client is nil")
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.RPush(r.Ctx, queue, b).Err()
}

func (r *RedisCache[T]) BRPop(timeout time.Duration, queue string) (value T, err error) {
	if !r.IsInitialized() {
		return value, fmt.Errorf("redis client is nil")
	}
	result, err := r.Client.BRPop(r.Ctx, timeout, queue).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return value, ErrCacheMiss
		}
		return value, err
	}

	if len(result) != 2 {
		return value, fmt.Errorf("invalid result length from BRPop for queue %s", queue)
	}

	if err = json.Unmarshal([]byte(result[1]), &value); err != nil {
		return value, err
	}
	return
}

func (r *RedisCache[T]) Incr(key string) (int64, error) {
	if !r.IsInitialized() {
		return 0, fmt.Errorf("redis client is nil")
	}
	return r.Client.Incr(r.Ctx, key).Result()
}

func (r *RedisCache[T]) Expire(key string, expiration time.Duration) error {
	if !r.IsInitialized() {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Expire(r.Ctx, key, expiration).Err()
}

func (r *RedisCache[T]) Flush() error {
	if !r.IsInitialized() {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.FlushAll(r.Ctx).Err()
}
