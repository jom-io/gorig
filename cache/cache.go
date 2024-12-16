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

// Cache 是一个通用的缓存接口，定义了基本的缓存操作
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

// ErrCacheMiss 表示缓存未命中的错误
var ErrCacheMiss = errors.New("cache miss")

// LoaderFunc 是一个用于从外部源加载数据的函数类型
type LoaderFunc[T any] func(key string) (T, error)

// Tool 是多级缓存的管理工具
type Tool[T any] struct {
	Ctx    *gin.Context
	caches []Cache[T]
	loader LoaderFunc[T]
	group  singleflight.Group
	mu     sync.Mutex
}

// NewCacheTool 创建一个新的 Tool 实例
func NewCacheTool[T any](ctx *gin.Context, caches []Cache[T], loader LoaderFunc[T]) *Tool[T] {
	return &Tool[T]{
		Ctx:    ctx,
		caches: caches,
		loader: loader,
	}
}

// Get 从缓存中获取数据，依次查找各级缓存，如果都未命中则通过 loader 加载
func (c *Tool[T]) Get(key string, expiration time.Duration) (T, error) {
	var zero T
	// 使用 singleflight 防止缓存击穿
	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		var value T

		// 从各级缓存中依次查找
		for i, cacheLayer := range c.caches {
			val, err := cacheLayer.Get(key)
			if err == nil {
				logger.Info(c.Ctx, fmt.Sprintf("Cache hit in layer %d", i+1))
				value = val
				// 将数据同步到更高级别的缓存
				for j := 0; j < i; j++ {
					err = c.caches[j].Set(key, value, expiration)
					if err != nil {
						return nil, err
					}
				}
				return value, nil
			}
		}

		// 如果所有缓存层都未命中，使用 loader 加载数据
		logger.Info(c.Ctx, "Cache miss in all layers, loading from external source")
		val, err := c.loader(key)
		if err != nil {
			return zero, err
		}
		value = val

		// 将数据存入所有缓存层
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

// Set 将数据存入所有缓存层
func (c *Tool[T]) Set(key string, value T, expiration time.Duration) error {
	for _, cacheLayer := range c.caches {
		if err := cacheLayer.Set(key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

// Delete 从所有缓存层中删除数据
func (c *Tool[T]) Delete(key string) error {
	for _, cacheLayer := range c.caches {
		if err := cacheLayer.Del(key); err != nil {
			return err
		}
	}
	return nil
}
