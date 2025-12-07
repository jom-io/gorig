package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteCache[T any] struct {
	db   *sql.DB
	lock sync.RWMutex
}

var (
	cacheSqliteIns sync.Map // map[string]any，缓存 SQLiteCache[T] 实例
	dbLock         sync.Mutex
	sqliteTimeOut  = 5 * time.Second
)

func NewSQLiteCache[T any](cacheType string) (*SQLiteCache[T], error) {
	dbLock.Lock()
	defer dbLock.Unlock()

	if val, ok := cacheSqliteIns.Load(cacheType); ok {
		if typed, ok := val.(*SQLiteCache[T]); ok {
			return typed, nil
		}
	}

	if err := os.MkdirAll(".cache", 0755); err != nil {
		return nil, err
	}
	dbPath := fmt.Sprintf(".cache/%s.db", cacheType)
	cleanupIfMissingBaseFile(dbPath)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil || err != nil {
			db.Close()
		}
	}()

	// open the database in WAL mode
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
		return nil, err
	}

	// create the cache table
	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS cache (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		expiration INTEGER
	);`); err != nil {
		db.Close()
		return nil, err
	}

	// create the queue table
	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS queue (
		key TEXT,
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		value TEXT NOT NULL
	);`); err != nil {
		db.Close()
		return nil, err
	}

	ins := &SQLiteCache[T]{db: db}
	cacheSqliteIns.Store(cacheType, ins)

	return ins, nil
}

func cleanupIfMissingBaseFile(dbPath string) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}
}

func (c *SQLiteCache[T]) IsInitialized() bool {
	return c != nil && c.db != nil
}

func (c *SQLiteCache[T]) Keys() ([]string, error) {
	if c == nil {
		return nil, errors.New("cache not initialized")
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	rows, err := c.db.Query("SELECT key, expiration FROM cache")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	now := time.Now().Unix()
	for rows.Next() {
		var key string
		var expiration int64
		if err := rows.Scan(&key, &expiration); err != nil {
			return nil, err
		}
		if expiration == 0 || now <= expiration {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (c *SQLiteCache[T]) Items() map[string]T {
	result := make(map[string]T)
	if c == nil {
		return result
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	rows, err := c.db.Query("SELECT key, value, expiration FROM cache")
	if err != nil {
		return result
	}
	defer rows.Close()

	now := time.Now().Unix()
	for rows.Next() {
		var key string
		var valueStr string
		var expiration int64
		if err := rows.Scan(&key, &valueStr, &expiration); err != nil {
			continue
		}
		if expiration > 0 && now > expiration {
			continue
		}
		var value T
		if err := json.Unmarshal([]byte(valueStr), &value); err == nil {
			result[key] = value
		}
	}
	return result
}

func (c *SQLiteCache[T]) Get(key string) (T, error) {
	var zero T
	if c == nil {
		return zero, errors.New("cache not initialized")
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	var valueStr string
	var expiration int64
	err := c.db.QueryRow("SELECT value, expiration FROM cache WHERE key = ?", key).Scan(&valueStr, &expiration)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return zero, nil
		}
		return zero, err
	}

	if expiration > 0 && time.Now().Unix() > expiration {
		c.Del(key)
		return zero, nil
	}

	err = json.Unmarshal([]byte(valueStr), &zero)
	return zero, err
}

func (c *SQLiteCache[T]) cleanup() {
	if c == nil {
		return
	}
	go func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
		defer cancel()
		now := time.Now().Unix()
		_, err := c.db.Exec("DELETE FROM cache WHERE expiration > 0 AND expiration < ?", now)
		if err != nil {
			fmt.Printf("Error cleaning up expired cache entries: %v\n", err)
		}
	}()
}

func (c *SQLiteCache[T]) Set(key string, value T, expiration time.Duration) error {
	if c == nil {
		return errors.New("cache not initialized")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	defer c.cleanup()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	exp := int64(0)
	if expiration > 0 {
		exp = time.Now().Add(expiration).Unix()
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = c.db.Exec(`INSERT OR REPLACE INTO cache(key, value, expiration) VALUES(?, ?, ?)`, key, string(b), exp)
	return err
}

func (c *SQLiteCache[T]) Del(key string) error {
	if c == nil {
		return errors.New("cache not initialized")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	defer c.cleanup()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	_, err := c.db.Exec("DELETE FROM cache WHERE key = ?", key)
	return err
}

func (c *SQLiteCache[T]) Exists(key string) (bool, error) {
	if c == nil {
		return false, errors.New("cache not initialized")
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	var expiration int64
	err := c.db.QueryRow("SELECT expiration FROM cache WHERE key = ?", key).Scan(&expiration)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if expiration > 0 && time.Now().Unix() > expiration {
		return false, nil
	}
	return true, nil
}

func (c *SQLiteCache[T]) Incr(key string) (int64, error) {
	if c == nil {
		return 0, errors.New("cache not initialized")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	defer c.cleanup()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	var valueStr string
	var expiration int64
	err := c.db.QueryRow("SELECT value, expiration FROM cache WHERE key = ?", key).Scan(&valueStr, &expiration)

	var curr int64
	if err == nil {
		var val any
		if json.Unmarshal([]byte(valueStr), &val) == nil {
			switch v := val.(type) {
			case float64:
				curr = int64(v)
			case int:
				curr = int64(v)
			case int64:
				curr = v
			}
		}
	}
	curr++
	newVal, _ := json.Marshal(curr)
	_, err = c.db.Exec("INSERT OR REPLACE INTO cache(key, value, expiration) VALUES(?, ?, ?)", key, string(newVal), expiration)
	return curr, err
}

func (c *SQLiteCache[T]) Expire(key string, expiration time.Duration) error {
	if c == nil {
		return errors.New("cache not initialized")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	defer c.cleanup()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	exp := int64(0)
	if expiration > 0 {
		exp = time.Now().Add(expiration).Unix()
	}
	_, err := c.db.Exec("UPDATE cache SET expiration = ? WHERE key = ?", exp, key)
	return err
}

func (c *SQLiteCache[T]) RPush(key string, value T) error {
	if c == nil {
		return errors.New("cache not initialized")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = c.db.Exec(`INSERT INTO queue (key, value) VALUES (?, ?)`, key, string(b))
	return err
}

func (c *SQLiteCache[T]) BRPop(sqliteTimeOut time.Duration, key string) (T, error) {
	var zero T
	if c == nil {
		return zero, errors.New("cache not initialized")
	}
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()
	start := time.Now()
	for {
		c.lock.Lock()
		row := c.db.QueryRow(`SELECT id, value FROM queue WHERE key = ? ORDER BY id LIMIT 1`, key)

		var id int64
		var valueStr string
		err := row.Scan(&id, &valueStr)
		if err == nil {
			_, _ = c.db.Exec(`DELETE FROM queue WHERE id = ?`, id)
			c.lock.Unlock()
			err = json.Unmarshal([]byte(valueStr), &zero)
			return zero, err
		}
		c.lock.Unlock()

		if time.Since(start) > sqliteTimeOut {
			return zero, ErrCacheMiss
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *SQLiteCache[T]) Flush() error {
	if c == nil {
		return errors.New("cache not initialized")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	_, err := c.db.Exec("DELETE FROM cache")
	return err
}
