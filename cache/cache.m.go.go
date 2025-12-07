package cache

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"sync"
	"time"
)

type Queue[T any] struct {
	Items []T
}

type GoCache[T any] struct {
	cache   *cache.Cache
	locks   sync.Map // map[string]*sync.RWMutex
	signals sync.Map // map[string]chan struct{}
}

func NewGoCache[T any](defaultExpiration, cleanupInterval time.Duration) *GoCache[T] {
	return &GoCache[T]{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

func (g *GoCache[T]) getLock(key string) *sync.RWMutex {
	actual, _ := g.locks.LoadOrStore(key, &sync.RWMutex{})
	return actual.(*sync.RWMutex)
}

func (g *GoCache[T]) getSignal(key string) chan struct{} {
	actual, _ := g.signals.LoadOrStore(key, make(chan struct{}, 1))
	return actual.(chan struct{})
}

func (g *GoCache[T]) IsInitialized() bool {
	return g != nil && g.cache != nil
}

func (g *GoCache[T]) Keys() ([]string, error) {
	items := g.cache.Items()
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	return keys, nil
}

func (g *GoCache[T]) Items() map[string]T {
	result := make(map[string]T)
	items := g.cache.Items()
	for k, v := range items {
		val, ok := v.Object.(T)
		if ok {
			result[k] = val
		}
	}
	return result
}

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

func (g *GoCache[T]) Set(key string, value T, expiration time.Duration) error {
	lock := g.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	g.cache.Set(key, value, expiration)
	return nil
}

func (g *GoCache[T]) Del(key string) error {
	lock := g.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	g.cache.Delete(key)
	g.locks.Delete(key)
	g.signals.Delete(key)
	return nil
}

func (g *GoCache[T]) Exists(key string) (bool, error) {
	lock := g.getLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if _, found := g.cache.Get(key); found {
		return true, nil
	}
	return false, nil
}

func (g *GoCache[T]) RPush(key string, value T) error {
	lock := g.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	queue, found := g.cache.Get(key)
	if !found {
		queue = &Queue[T]{}
	}
	q, ok := queue.(*Queue[T])
	if !ok {
		return fmt.Errorf("type assertion failed for key %s", key)
	}
	q.Items = append(q.Items, value)
	g.cache.Set(key, q, cache.NoExpiration)

	signal := g.getSignal(key)
	select {
	case signal <- struct{}{}:
	default:
	}

	return nil
}

func (g *GoCache[T]) BRPop(timeout time.Duration, key string) (T, error) {
	var zero T

	for {
		lock := g.getLock(key)
		lock.Lock()

		queue, found := g.cache.Get(key)
		if found {
			q, ok := queue.(*Queue[T])
			if !ok {
				lock.Unlock()
				return zero, fmt.Errorf("type assertion failed for key %s", key)
			}
			if len(q.Items) > 0 {
				value := q.Items[0]
				q.Items = q.Items[1:]
				g.cache.Set(key, q, cache.NoExpiration)
				lock.Unlock()
				return value, nil
			}
		}

		signal := g.getSignal(key)
		if timeout > 0 {
			timer := time.NewTimer(timeout)
			lock.Unlock()

			select {
			case <-signal:
				// has signal
			case <-timer.C:
				// timeout
				return zero, ErrCacheMiss
			}
		} else {
			// wait forever
			lock.Unlock()
			<-signal
		}
	}
}

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

func (g *GoCache[T]) Flush() error {
	g.cache.Flush()
	return nil
}
