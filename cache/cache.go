package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/jom-io/gorig/utils/logger"
	"golang.org/x/sync/singleflight"
	"path/filepath"
	"sync"
	"time"
)

// Cache is a generic cache interface that defines basic cache operations
type Cache[T any] interface {
	IsInitialized() bool
	Keys() ([]string, error)
	Items() map[string]T
	Get(key string) (T, error)
	Set(key string, value T, expiration time.Duration) error
	Del(key string) error
	Exists(key string) (bool, error)
	RPush(key string, value T) error
	BRPop(timeout time.Duration, key string) (value T, err error)
	Incr(key string) (int64, error)
	Expire(key string, expiration time.Duration) error
	Flush() error
}

type Type string

const (
	Memory Type = "memory"
	Redis  Type = "redis"
	JSON   Type = "json"
	Sqlite Type = "sqlite"
)

func New[T any](t Type, args ...any) Cache[T] {
	switch t {
	case Memory:
		var defaultExpiration, cleanupInterval = time.Minute, time.Minute
		if len(args) == 1 {
			defaultExpiration = args[0].(time.Duration)
		} else if len(args) == 2 {
			defaultExpiration = args[0].(time.Duration)
			cleanupInterval = args[1].(time.Duration)
		}
		return NewGoCache[T](defaultExpiration, cleanupInterval)
	case Redis:
		return GetRedisInstance[T](context.Background())
	case JSON:
		if len(args) < 1 {
			args = append(args, filepath.Base(fmt.Sprintf("%T", new(T))))
		}
		cache, err := NewJSONCache[T](args[0].(string))
		if err != nil {
			logger.Error(nil, fmt.Sprintf("Failed to create JSON cache: %v", err))
		}
		return cache
	case Sqlite:
		if len(args) < 1 {
			args = append(args, filepath.Base(fmt.Sprintf("%T", new(T))))
		}
		cache, err := NewSQLiteCache[T](args[0].(string))
		if err != nil {
			logger.Error(nil, fmt.Sprintf("Failed to create SQLite cache: %v", err))
		}
		return cache
	default:
		logger.Error(nil, fmt.Sprintf("Unsupported cache type: %s, using memory cache", t))
		return NewGoCache[T](time.Minute, time.Minute)
	}
}

// ErrCacheMiss indicates a cache miss error
var ErrCacheMiss = errors.New("cache miss")

// LoaderFunc is a function type for loading data from an external source
type LoaderFunc[T any] func(key string) (T, error)

// Tool is a management tool for multi-level cache
type Tool[T any] struct {
	Ctx    context.Context
	caches []Cache[T]
	loader LoaderFunc[T]
	group  singleflight.Group
	mu     sync.Mutex
}

// NewCacheTool creates a new Tool instance
func NewCacheTool[T any](ctx context.Context, caches []Cache[T], loader LoaderFunc[T]) *Tool[T] {
	return &Tool[T]{
		Ctx:    ctx,
		caches: caches,
		loader: loader,
	}
}

// Get retrieves data from the cache, searching each level in order, and loads from the loader if all levels miss
func (c *Tool[T]) Get(key string, expiration time.Duration) (T, error) {
	var zero T
	// Use singleflight to prevent cache stampede
	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		var value T

		// Search each cache level in order
		for i, cacheLayer := range c.caches {
			val, err := cacheLayer.Get(key)
			if err == nil {
				logger.Info(c.Ctx, fmt.Sprintf("Cache hit in layer %d", i+1))
				value = val
				// Sync data to higher-level caches
				for j := 0; j < i; j++ {
					err = c.caches[j].Set(key, value, expiration)
					if err != nil {
						return nil, err
					}
				}
				return value, nil
			}
		}

		// If no loader, return cacheMiss directly
		if c.loader == nil {
			return zero, ErrCacheMiss
		}

		// If all cache levels miss, load data using loader
		logger.Info(c.Ctx, "Cache miss in all layers, loading from external source")
		val, err := c.loader(key)
		if err != nil {
			return zero, err
		}
		value = val

		// Store data in all cache levels
		for _, cacheLayer := range c.caches {
			cacheLayer.Set(key, value, expiration)
		}

		return value, nil
	})

	if err != nil {
		return zero, err
	}

	return v.(T), nil
}

// Set stores data in all cache levels
func (c *Tool[T]) Set(key string, value T, expiration time.Duration) error {
	for _, cacheLayer := range c.caches {
		if err := cacheLayer.Set(key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes data from all cache levels
func (c *Tool[T]) Delete(key string) error {
	for _, cacheLayer := range c.caches {
		if err := cacheLayer.Del(key); err != nil {
			return err
		}
	}
	return nil
}
