package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type jsonCacheItem[T any] struct {
	Value      T     `json:"value"`
	Expiration int64 `json:"expiration"`
}

type JSONFileCache[T any] struct {
	filePath string
	data     map[string]jsonCacheItem[T]
	lock     sync.RWMutex
}

func NewJSONCache[T any](cacheType string) (*JSONFileCache[T], error) {
	dir := ".cache"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(dir, fmt.Sprintf("%s.cache.json", cacheType))

	cache := &JSONFileCache[T]{
		filePath: filePath,
		data:     make(map[string]jsonCacheItem[T]),
	}
	err := cache.loadFromFile()
	return cache, err
}

func (c *JSONFileCache[T]) IsInitialized() bool {
	return c != nil && c.data != nil
}

func (c *JSONFileCache[T]) loadFromFile() error {
	file, err := os.Open(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&c.data)
}

func (c *JSONFileCache[T]) saveToFile() error {
	file, err := os.Create(c.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(c.data)
}

func (c *JSONFileCache[T]) cleanup() {
	now := time.Now().Unix()
	for k, v := range c.data {
		if v.Expiration > 0 && now > v.Expiration {
			delete(c.data, k)
		}
	}
}

func (c *JSONFileCache[T]) Get(key string) (T, error) {
	var zero T
	c.lock.RLock()
	defer c.lock.RUnlock()

	item, found := c.data[key]
	if !found || (item.Expiration > 0 && time.Now().Unix() > item.Expiration) {
		return zero, ErrCacheMiss
	}
	return item.Value, nil
}

func (c *JSONFileCache[T]) Set(key string, value T, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	exp := int64(0)
	if expiration > 0 {
		exp = time.Now().Add(expiration).Unix()
	}
	c.data[key] = jsonCacheItem[T]{Value: value, Expiration: exp}
	c.cleanup()
	return c.saveToFile()
}

func (c *JSONFileCache[T]) Del(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.data, key)
	c.cleanup()
	return c.saveToFile()
}

func (c *JSONFileCache[T]) Exists(key string) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	item, found := c.data[key]
	if !found || (item.Expiration > 0 && time.Now().Unix() > item.Expiration) {
		return false, nil
	}
	return true, nil
}

func (c *JSONFileCache[T]) Incr(key string) (int64, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var curr int64
	item, found := c.data[key]
	if found {
		switch v := any(item.Value).(type) {
		case float64:
			curr = int64(v)
		case int:
			curr = int64(v)
		case int64:
			curr = v
		default:
			return 0, fmt.Errorf("invalid type for Incr key %s", key)
		}
	}
	curr++
	c.data[key] = jsonCacheItem[T]{Value: any(curr).(T), Expiration: item.Expiration}
	c.cleanup()
	return curr, c.saveToFile()
}

func (c *JSONFileCache[T]) Expire(key string, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, found := c.data[key]
	if !found {
		return ErrCacheMiss
	}
	if expiration > 0 {
		item.Expiration = time.Now().Add(expiration).Unix()
	} else {
		item.Expiration = 0
	}
	c.data[key] = item
	c.cleanup()
	return c.saveToFile()
}

func (c *JSONFileCache[T]) RPush(key string, value T) error {
	return errors.New("RPush not supported in file cache")
}

func (c *JSONFileCache[T]) BRPop(timeout time.Duration, key string) (value T, err error) {
	return value, errors.New("BRPop not supported in file cache")
}

func (c *JSONFileCache[T]) Flush() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := os.Remove(c.filePath); err != nil {
		return err
	}
	c.data = make(map[string]jsonCacheItem[T])
	return nil
}
