// cache/go_cache.go
package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/sys"
)

var (
	RedisInstance *RedisCache
	initMu        sync.Mutex
)

func RestRedisInstance() {
	initRedisCache()
}

// GetRedisInstance 返回单例的 RedisCache 实例
func GetRedisInstance(r ...*RedisCache) *RedisCache {
	initMu.Lock()
	defer initMu.Unlock()
	// 再次检查是否已经初始化
	if RedisInstance == nil {
		initRedisCache()
	}
	if len(r) > 0 {
		r[0] = RedisInstance
	}
	return RedisInstance
}

// initRedisCache 初始化 RedisCache
func initRedisCache() {
	RedisInstance = nil
	addr := configure.GetString("redis.addr")
	password := configure.GetString("redis.password")
	db := configure.GetInt("redis.db")
	if addr == "" {
		sys.Error("# Redis addr is empty, skipping initialization")
		return
	}

	cache, err := NewRedisCache(RedisConfig{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err != nil {
		sys.Error("# failed to init Redis cache: ", err)
		return
	}
	RedisInstance = cache
	sys.Info("# Redis cache initialized")
}

// RedisConfig holds the Redis configuration parameters
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// RedisCache 是 Redis 的实现
type RedisCache struct {
	Client *redis.Client
	Ctx    context.Context
}

// NewRedisCache 创建一个新的 RedisCache 实例
func NewRedisCache(cfg RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisCache{
		Client: client,
		Ctx:    ctx,
	}, nil
}

// IsInitialized bool
func (r *RedisCache) IsInitialized() bool {
	return r != nil && r.Client != nil
}

// Get 从 Redis 中获取数据
func (r *RedisCache) Get(key string) (interface{}, error) {
	if GetRedisInstance(r) == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	val, err := r.Client.Get(r.Ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	} else if err != nil {
		return nil, err
	}
	return val, nil
}

// Set 将数据存入 Redis
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	if GetRedisInstance(r) == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Set(r.Ctx, key, value, expiration).Err()
}

// Del 从 Redis 中删除数据
func (r *RedisCache) Del(key string) error {
	if GetRedisInstance() == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Del(r.Ctx, key).Err()
}

// Exists 检查 Redis 中是否存在指定 key
func (r *RedisCache) Exists(key string) (bool, error) {
	if GetRedisInstance(r) == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	result, err := r.Client.Exists(r.Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// RPush 将数据存入 Redis
func (r *RedisCache) RPush(key string, value interface{}) error {
	if GetRedisInstance(r) == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.RPush(r.Ctx, key, value).Err()
}

// BRPop 从 Redis 中弹出数据 timeout 为超时时间 key 为队列名
func (r *RedisCache) BRPop(timeout time.Duration, key string) (value interface{}, err error) {
	if GetRedisInstance(r) == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	result, err := r.Client.BRPop(r.Ctx, timeout, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	if len(result) != 2 {
		return nil, fmt.Errorf("invalid result length from BRPop for key %s", key)
	}
	return result[1], nil
}

// Incr 递增 Redis 中的值
func (r *RedisCache) Incr(key string) (int64, error) {
	if GetRedisInstance(r) == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return r.Client.Incr(r.Ctx, key).Result()
}

// Expire 设置 Redis 中 key 的过期时间
func (r *RedisCache) Expire(key string, expiration time.Duration) error {
	if GetRedisInstance(r) == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Expire(r.Ctx, key, expiration).Err()
}