package cache

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/sys"
	"time"
)

var RedisInstance *RedisCache

func GetRedisInstance() *RedisCache {
	if RedisInstance == nil {
		initRedisCache()
	}
	return RedisInstance
}

// InitRedisCache 初始化 RedisCache
func initRedisCache() {
	addr := configure.GetString("redis.addr")
	password := configure.GetString("redis.password")
	db := configure.GetInt("redis.db")
	if addr == "" {
		return
	}

	cache, err := NewRedisCache(RedisConfig{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err != nil {
		if err != nil {
			sys.Error("# failed to init Redis cache: %v", err)
		} else {
			sys.Info("# Redis cache initialized")
		}
		return
	}
	RedisInstance = cache
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

// Get 从 Redis 中获取数据
func (r *RedisCache) Get(key string) (interface{}, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("Redis client is nil")
	}
	val, err := r.Client.Get(r.Ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss in Redis")
	} else if err != nil {
		return nil, err
	}
	return val, nil
}

// Set 将数据存入 Redis
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("Redis client is nil")
	}
	return r.Client.Set(r.Ctx, key, value, expiration).Err()
}

// Delete 从 Redis 中删除数据
func (r *RedisCache) Delete(key string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("Redis client is nil")
	}
	return r.Client.Del(r.Ctx, key).Err()
}
