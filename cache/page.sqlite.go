package cache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type SQLiteCachePage[T any] struct {
	dbPath string
	db     *sql.DB
	table  string
	//once   sync.Once
	mu sync.RWMutex
}

var pageNewLock sync.Mutex

// NewSQLiteCachePage Create a new SQLite cache page with the given name.
func NewSQLiteCachePage[T any](name string) (*SQLiteCachePage[T], error) {
	pageNewLock.Lock()
	defer pageNewLock.Unlock()
	if err := os.MkdirAll(".cache", 0755); err != nil {
		return nil, err
	}
	dbPath := fmt.Sprintf(".cache/%s.pg.db", name)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	// Set the SQLite journal mode to WAL (Write-Ahead Logging)
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return nil, err
	}

	//db.SetMaxOpenConns(4)

	table := strings.ToLower(name)
	table = strings.ReplaceAll(table, "*", "")
	table = strings.ReplaceAll(table, " ", "_")
	table = strings.ReplaceAll(table, "-", "_")
	table = strings.ReplaceAll(table, ".", "_")
	table = strings.ReplaceAll(table, "/", "_")
	table = strings.ReplaceAll(table, "\\", "_")
	cache := &SQLiteCachePage[T]{dbPath: dbPath, db: db, table: table}

	if err := cache.ensureTable(); err != nil {
		return nil, err
	}

	return cache, nil
}

func (c *SQLiteCachePage[T]) ensureTable() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		data TEXT
	);`, c.table)
	_, err := c.db.Exec(query)
	return err
}

func (c *SQLiteCachePage[T]) Put(value T) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = c.db.Exec(fmt.Sprintf(`INSERT INTO %s (data) VALUES (?)`, c.table), string(bytes))
	return err
}

func (c *SQLiteCachePage[T]) Count(conditions map[string]any) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	where, args := buildWhereClause(conditions)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s %s`, c.table, where)

	var count int64
	err := c.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (c *SQLiteCachePage[T]) Get(conditions map[string]any) (*T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	where, args := buildWhereClause(conditions)
	query := fmt.Sprintf(`SELECT data FROM %s %s`, c.table, where)

	var jsonStr string
	err := c.db.QueryRow(query, args...).Scan(&jsonStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var item T
	if err := json.Unmarshal([]byte(jsonStr), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *SQLiteCachePage[T]) Find(page, size int64, conditions map[string]any, sorts ...PageSorter) (*PageCache[T], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * size
	where, args := buildWhereClause(conditions)

	orderBy := "ORDER BY id DESC"
	//orderBy = fmt.Sprintf("ORDER BY json_extract(data, '$.%s') %s", sort.SortField, desc)
	if len(sorts) > 0 {
		orderClauses := make([]string, len(sorts))
		for i, sort := range sorts {
			if sort.Asc {
				orderClauses[i] = fmt.Sprintf("json_extract(data, '$.%s') ASC", sort.SortField)
			} else {
				orderClauses[i] = fmt.Sprintf("json_extract(data, '$.%s') DESC", sort.SortField)
			}
		}
		orderBy = "ORDER BY " + strings.Join(orderClauses, ", ")
	}

	query := fmt.Sprintf(`SELECT data FROM %s %s %s LIMIT ? OFFSET ?`, c.table, where, orderBy)
	args = append(args, size, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	count, err := c.Count(conditions)

	var results []*T
	for rows.Next() {
		var jsonStr string
		if err := rows.Scan(&jsonStr); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal([]byte(jsonStr), &item); err != nil {
			return nil, err
		}
		results = append(results, &item)
	}
	return &PageCache[T]{Total: count, Page: page, Size: size, Items: results}, nil
}

func (c *SQLiteCachePage[T]) Update(conditions map[string]any, value *T) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(conditions) == 0 {
		return fmt.Errorf("conditions cannot be empty")
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	upArgs := []any{string(bytes)}

	where, args := buildWhereClause(conditions)
	upArgs = append(upArgs, args...)

	query := fmt.Sprintf(`UPDATE %s SET data = ? %s `, c.table, where)

	_, err = c.db.Exec(query, upArgs...)
	if err != nil {
		//logger.Error(nil, fmt.Sprintf("Failed to update SQLite cache: %v", err))
	}
	return err
}

func buildWhereClause(conditions map[string]any) (string, []any) {
	if len(conditions) == 0 {
		return "", nil
	}
	where := "WHERE "
	args := make([]any, 0, len(conditions))
	clauses := make([]string, 0, len(conditions))
	for k, v := range conditions {
		clauses = append(clauses, fmt.Sprintf("json_extract(data, '$.%s') = ?", k))
		args = append(args, v)
	}
	where += strings.Join(clauses, " AND ")
	return where, args
}
