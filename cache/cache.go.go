package cache

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"sync"
	"time"
)

// Queue 定义一个泛型队列
type Queue[T any] struct {
	Items []T
}

// GoCache 是 go-cache 的实现，支持泛型，并使用通道进行阻塞操作
type GoCache[T any] struct {
	cache   *cache.Cache
	locks   sync.Map // map[string]*sync.RWMutex
	signals sync.Map // map[string]chan struct{}
}

// NewGoCache 创建一个新的 GoCache 实例
// defaultExpiration: 默认过期时间
// cleanupInterval: 清理间隔
func NewGoCache[T any](defaultExpiration, cleanupInterval time.Duration) *GoCache[T] {
	return &GoCache[T]{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// getLock 获取或创建与 key 关联的锁
func (g *GoCache[T]) getLock(key string) *sync.RWMutex {
	actual, _ := g.locks.LoadOrStore(key, &sync.RWMutex{})
	return actual.(*sync.RWMutex)
}

// getSignal 获取或创建与 key 关联的信号通道
func (g *GoCache[T]) getSignal(key string) chan struct{} {
	actual, _ := g.signals.LoadOrStore(key, make(chan struct{}, 1))
	return actual.(chan struct{})
}

// IsInitialized 检查缓存是否已初始化
func (g *GoCache[T]) IsInitialized() bool {
	return g != nil && g.cache != nil
}

// Get 从 go-cache 中获取数据
func (g *GoCache[T]) Get(key string) (T, error) {
	var zero T
	lock := g.getLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if data, found := g.cache.Get(key); found {
		val, ok := data.(T)
		if !ok {
			return zero, fmt.Errorf("type assertion failed for key %s", key)
		}
		return val, nil
	}
	return zero, ErrCacheMiss
}

// Set 将数据存入 go-cache
func (g *GoCache[T]) Set(key string, value T, expiration time.Duration) error {
	lock := g.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	g.cache.Set(key, value, expiration)
	return nil
}

// Del 从 go-cache 中删除数据
func (g *GoCache[T]) Del(key string) error {
	lock := g.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	g.cache.Delete(key)
	g.locks.Delete(key)   // 删除锁，防止锁映射表无限增长
	g.signals.Delete(key) // 删除信号通道，防止映射表无限增长
	return nil
}

// Exists 检查 key 是否存在
func (g *GoCache[T]) Exists(key string) (bool, error) {
	lock := g.getLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if _, found := g.cache.Get(key); found {
		return true, nil
	}
	return false, nil
}

// RPush 将数据存入 go-cache
func (g *GoCache[T]) RPush(key string, value T) error {
	lock := g.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	// 获取队列
	queue, found := g.cache.Get(key)
	if !found {
		queue = &Queue[T]{}
	}
	q, ok := queue.(*Queue[T])
	if !ok {
		return fmt.Errorf("type assertion failed for key %s", key)
	}
	// 添加数据
	q.Items = append(q.Items, value)
	g.cache.Set(key, q, cache.NoExpiration)

	// 唤醒等待的 BRPop
	signal := g.getSignal(key)
	select {
	case signal <- struct{}{}:
	default:
		// 如果信号通道已满，不阻塞
	}

	return nil
}

// BRPop 从 go-cache 中弹出数据，支持阻塞和超时
func (g *GoCache[T]) BRPop(timeout time.Duration, key string) (T, error) {
	var zero T

	for {
		lock := g.getLock(key)
		lock.Lock()

		// 获取队列
		queue, found := g.cache.Get(key)
		if found {
			q, ok := queue.(*Queue[T])
			if !ok {
				lock.Unlock()
				return zero, fmt.Errorf("type assertion failed for key %s", key)
			}
			if len(q.Items) > 0 {
				// 弹出数据
				value := q.Items[0]
				q.Items = q.Items[1:]
				g.cache.Set(key, q, cache.NoExpiration)
				lock.Unlock()
				return value, nil
			}
		}

		// 队列为空，等待通知或超时
		signal := g.getSignal(key)
		if timeout > 0 {
			timer := time.NewTimer(timeout)
			lock.Unlock()

			select {
			case <-signal:
				// 有新数据，继续循环
			case <-timer.C:
				// 超时
				return zero, ErrCacheMiss
			}
		} else {
			// 无限等待
			lock.Unlock()
			<-signal
		}
	}
}

// Incr 递增 key 的值
func (g *GoCache[T]) Incr(key string) (int64, error) {
	lock := g.getLock("incr" + key)
	lock.Lock()
	defer lock.Unlock()

	val, found := g.cache.Get(key)
	if !found {
		val = int64(0)
	}
	v, ok := any(val).(int64)
	if !ok {
		return 0, fmt.Errorf("type assertion failed for key %s", key)
	}
	v++
	g.cache.Set(key, v, cache.NoExpiration)
	return v, nil
}

// Expire 设置 key 的过期时间
func (g *GoCache[T]) Expire(key string, expiration time.Duration) error {
	lock := g.getLock("expire" + key)
	lock.Lock()
	defer lock.Unlock()

	val, err := g.Get(key)
	if err != nil {
		return err
	}
	g.cache.Set(key, val, expiration)
	return nil
}
