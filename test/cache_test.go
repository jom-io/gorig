package test

import (
	"database/sql"
	"github.com/jom-io/gorig/cache"
	"testing"
	"time"
)

func TestJSONFileCache_BasicOperations(t *testing.T) {
	type User struct {
		Name string
		Age  int
	}

	cacheIns := cache.New[User](cache.JSON, "test_user_cache")
	//if err != nil {
	//	t.Fatalf("failed to create cache: %v", err)
	//}

	defer func() {
		_ = cacheIns.Flush()
	}()

	err := cacheIns.Set("user1", User{Name: "Alice", Age: 30}, 2*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	user, err := cacheIns.Get("user1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if user.Name != "Alice" || user.Age != 30 {
		t.Errorf("unexpected user data: %+v", user)
	}

	ok, _ := cacheIns.Exists("user1")
	if !ok {
		t.Errorf("Exists returned false, expected true")
	}

	time.Sleep(3 * time.Second)

	_, err = cacheIns.Get("user1")
	if err == nil {
		t.Errorf("expected cache miss after expiration, got value")
	}

	err = cacheIns.Set("user2", User{Name: "Bob", Age: 40}, 0) // 永不过期
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = cacheIns.Del("user2")
	if err != nil {
		t.Errorf("Del failed: %v", err)
	}

	ok, _ = cacheIns.Exists("user2")
	if ok {
		t.Errorf("Expected false after Del, got true")
	}
}

func TestJSONFileCache_IncrAndExpire(t *testing.T) {
	//cacheIns, err := cache.NewJSONCache[int64]("test_incr_cache")
	cacheIns := cache.New[int64](cache.JSON, "test_user_cache")

	defer func() {
		_ = cacheIns.Flush()
	}()

	val, err := cacheIns.Incr("counter")
	if err != nil || val != 1 {
		t.Errorf("Incr expected 1, got %d, err: %v", val, err)
	}

	val, err = cacheIns.Incr("counter")
	if err != nil || val != 2 {
		t.Errorf("Incr expected 2, got %d, err: %v", val, err)
	}

	err = cacheIns.Expire("counter", 1*time.Second)
	if err != nil {
		t.Errorf("Expire failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = cacheIns.Get("counter")
	if err == nil {
		t.Errorf("Expected expired key to return error")
	}
}

func TestSQLiteCachePage_IndexTags(t *testing.T) {
	type Stat struct {
		At     int64  `json:"at" idx:"at"`
		Method string `json:"method" idx_group:"method_uri"`
		URI    string `json:"uri" idx_group:"method_uri"`
		Count  int64  `json:"count"`
	}

	name := "idx_test_cache"
	_, err := cache.NewSQLiteCachePage[Stat](name)
	if err != nil {
		t.Fatalf("NewSQLiteCachePage failed: %v", err)
	}

	dbPath := ".cache/" + name + ".pg.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("PRAGMA index_list(" + name + ");")
	if err != nil {
		t.Fatalf("query index_list failed: %v", err)
	}
	defer rows.Close()

	expected := map[string]bool{
		"idx_" + name + "_at":         false,
		"idx_" + name + "_method_uri": false,
	}

	for rows.Next() {
		var seq int
		var idxName string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &idxName, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index_list failed: %v", err)
		}
		if _, ok := expected[idxName]; ok {
			expected[idxName] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("index_list rows error: %v", err)
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing index: %s", name)
		}
	}
}
