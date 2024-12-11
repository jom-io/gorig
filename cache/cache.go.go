// cache/go_cache.go
package cache

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"time"
)

// GoCache 是 go-cache 的实现
type GoCache[T any] struct {
	cache *cache.Cache
}

// NewGoCache 创建一个新的 GoCache 实例 defaultExpiration: 默认过期时间 cleanupInterval: 清理间隔
func NewGoCache[T any](defaultExpiration, cleanupInterval time.Duration) *GoCache[T] {
	return &GoCache[T]{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Get 从 go-cache 中获取数据
func (g *GoCache[T]) Get(key string) (T, error) {
	var zero T
	if data, found := g.cache.Get(key); found {
		// 断言类型
		val, ok := data.(T)
		if !ok {
			return zero, fmt.Errorf("type assertion failed for key %s", key)
		}
		return val, nil
	}
	return zero, fmt.Errorf("cache miss in GoCache for key: %s", key)
}

// Set 将数据存入 go-cache
func (g *GoCache[T]) Set(key string, value T, expiration time.Duration) error {
	g.cache.Set(key, value, expiration)
	return nil
}

// Delete 从 go-cache 中删除数据
func (g *GoCache[T]) Delete(key string) error {
	g.cache.Delete(key)
	return nil
}
