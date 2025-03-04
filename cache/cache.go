package cache

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/utils/logger"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache is a generic cache interface that defines basic cache operations
type Cache[T any] interface {
	IsInitialized() bool
	Get(key string) (T, error)
	Set(key string, value T, expiration time.Duration) error
	Del(key string) error
	Exists(key string) (bool, error)
	RPush(key string, value T) error
	BRPop(timeout time.Duration, key string) (value interface{}, err error)
	Incr(key string) (int64, error)
	Expire(key string, expiration time.Duration) error
}

// ErrCacheMiss indicates a cache miss error
var ErrCacheMiss = errors.New("cache miss")

// LoaderFunc is a function type for loading data from an external source
type LoaderFunc[T any] func(key string) (T, error)

// Tool is a management tool for multi-level cache
type Tool[T any] struct {
	Ctx    *gin.Context
	caches []Cache[T]
	loader LoaderFunc[T]
	group  singleflight.Group
	mu     sync.Mutex
}

// NewCacheTool creates a new Tool instance
func NewCacheTool[T any](ctx *gin.Context, caches []Cache[T], loader LoaderFunc[T]) *Tool[T] {
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
