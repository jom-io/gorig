package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jom-io/gorig/utils/decimal"
	"github.com/jom-io/gorig/utils/logger"
	_ "modernc.org/sqlite"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SQLiteCachePage[T any] struct {
	dbPath string
	db     *sql.DB
	table  string
	mu     sync.RWMutex
}

var (
	cachePageSqliteIns sync.Map
	dbPageLock         sync.Mutex
)

var granularityFormats = map[Granularity]string{
	GranularityMinute:    "%Y-%m-%d %H:%M",
	GranularityHour:      "%Y-%m-%d %H",
	GranularityDay:       "%Y-%m-%d",
	GranularityWeek:      "%Y-%W",
	GranularityMonth:     "%Y-%m",
	GranularityYear:      "%Y",
	Granularity5Minutes:  "%Y-%m-%d %H:%M",
	Granularity10Minutes: "%Y-%m-%d %H:%M",
	Granularity30Minutes: "%Y-%m-%d %H:%M",
}

// NewSQLiteCachePage Create a new SQLite cache page with the given name.
func NewSQLiteCachePage[T any](name string) (*SQLiteCachePage[T], error) {
	dbPageLock.Lock()
	defer dbPageLock.Unlock()

	if val, ok := cachePageSqliteIns.Load(name); ok {
		if typed, ok := val.(*SQLiteCachePage[T]); ok {
			return typed, nil
		}
	}

	if err := os.MkdirAll(".cache", 0755); err != nil {
		return nil, err
	}
	dbPath := fmt.Sprintf(".cache/%s.pg.db", name)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil || err != nil {
			db.Close()
		}
	}()

	// Set the SQLite journal mode to WAL (Write-Ahead Logging)
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
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
		db.Close()
		return nil, err
	}

	cachePageSqliteIns.Store(name, cache)

	return cache, nil
}

func (c *SQLiteCachePage[T]) ensureColumn(ctx context.Context, column string) error {
	query := fmt.Sprintf("PRAGMA table_info(%s);", c.table)
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var exists bool
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			exists = true
			break
		}
	}

	if !exists {
		addSQL := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s TIMESTAMP;`, c.table, column)
		if _, err := c.db.ExecContext(ctx, addSQL); err != nil {
			return fmt.Errorf("add column %s failed: %w", column, err)
		}
	}
	return nil
}

func (c *SQLiteCachePage[T]) ensureTable() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()
	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		data TEXT,
		ct TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		ut TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`, c.table)

	if _, err := c.db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("create table failed: %w", err)
	}

	if err := c.ensureColumn(ctx, "ct"); err != nil {
		return fmt.Errorf("ensure column ct failed: %w", err)
	}

	if err := c.ensureColumn(ctx, "ut"); err != nil {
		return fmt.Errorf("ensure column ut failed: %w", err)
	}

	indexSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_ct ON %s(ct);`, c.table, c.table)
	if _, err := c.db.ExecContext(ctx, indexSQL); err != nil {
		return fmt.Errorf("create index failed: %w", err)
	}

	triggerSQL := fmt.Sprintf(`
	CREATE TRIGGER IF NOT EXISTS trg_%s_ut
	AFTER UPDATE ON %s
	FOR EACH ROW
	BEGIN
		UPDATE %s SET ut = CURRENT_TIMESTAMP WHERE id = OLD.id;
	END;`, c.table, c.table, c.table)

	if _, err := c.db.ExecContext(ctx, triggerSQL); err != nil {
		return fmt.Errorf("create trigger failed: %w", err)
	}
	return nil
}

func (c *SQLiteCachePage[T]) Put(value T) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = c.db.Exec(fmt.Sprintf(`INSERT INTO %s (data, ct) VALUES (?, CURRENT_TIMESTAMP)`, c.table), string(bytes))
	return err
}

func (c *SQLiteCachePage[T]) Count(conditions map[string]any) (int64, error) {
	//c.mu.RLock()
	//defer c.mu.RUnlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	where, args := buildWhereClause(conditions)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s %s`, c.table, where)

	var count int64
	err := c.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (c *SQLiteCachePage[T]) Get(conditions map[string]any) (*T, error) {
	//c.mu.RLock()
	//defer c.mu.RUnlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

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
	//c.mu.RLock()
	//defer c.mu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * size
	where, args := buildWhereClause(conditions)

	//orderBy = fmt.Sprintf("ORDER BY json_extract(data, '$.%s') %s", sort.SortField, desc)
	orderBy := getOrderByClause(sorts)

	query := fmt.Sprintf(`SELECT data FROM %s %s %s LIMIT ? OFFSET ?`, c.table, where, orderBy)
	args = append(args, size, offset)

	rows, err := c.db.QueryContext(ctx, query, args...)
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

func getOrderByClause(sorts []PageSorter) string {
	if len(sorts) == 0 {
		return "ORDER BY id DESC"
	}

	orderClauses := make([]string, len(sorts))
	for i, sort := range sorts {
		if sort.Asc {
			orderClauses[i] = fmt.Sprintf("json_extract(data, '$.%s') ASC", sort.SortField)
		} else {
			orderClauses[i] = fmt.Sprintf("json_extract(data, '$.%s') DESC", sort.SortField)
		}
	}
	return "ORDER BY " + strings.Join(orderClauses, ", ")
}

func (c *SQLiteCachePage[T]) GroupByTime(
	conditions map[string]any,
	from, to time.Time,
	granularity Granularity,
	agg Agg,
	fields ...string,
) ([]*PageTimeItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	timeFormat, ok := granularityFormats[granularity]
	if !ok {
		return nil, fmt.Errorf("unsupported granularity: %s", granularity)
	}

	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("from and to times must be provided")
	}

	if from.After(to) {
		return nil, fmt.Errorf("from time cannot be after to time")
	}

	where, args := buildWhereClause(conditions)

	from = from.UTC()
	to = to.UTC()
	if len(args) > 0 {
		where = fmt.Sprintf("%s AND ct BETWEEN ? AND ?", where)
		args = append(args, from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"))
	} else {
		where = fmt.Sprintf("WHERE ct BETWEEN ? AND ?")
		args = []any{from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05")}
	}
	orderBy := "ORDER BY ct ASC"

	aggFields := make([]string, 0, len(fields))
	aggFieldNames := make([]string, len(fields))
	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one field must be specified for aggregation")
	}
	for i, field := range fields {
		alias := fmt.Sprintf("agg_%s", field)
		aggFields = append(aggFields, fmt.Sprintf("%s(CAST(json_extract(data, '$.%s') AS REAL)) as %s", agg, field, alias))
		aggFieldNames[i] = field
	}

	timeFmt := fmt.Sprintf("strftime('%s', ct)", timeFormat)
	if granularity == Granularity5Minutes {
		timeFmt = "strftime('%Y-%m-%d %H:%M', datetime((strftime('%s', ct)/300)*300, 'unixepoch'))"
	}
	if granularity == Granularity10Minutes {
		timeFmt = "strftime('%Y-%m-%d %H:%M', datetime((strftime('%s', ct)/600)*600, 'unixepoch'))"
	}
	if granularity == Granularity30Minutes {
		timeFmt = "strftime('%Y-%m-%d %H:%M', datetime((strftime('%s', ct)/1800)*1800, 'unixepoch'))"
	}

	aggFieldsStr := strings.Join(aggFields, ", ")
	query := fmt.Sprintf(`SELECT %s as grp, %s FROM %s %s GROUP BY grp %s`, timeFmt, aggFieldsStr, c.table, where, orderBy)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	result := make([]*PageTimeItem, 0)
	for rows.Next() {
		cols := make([]interface{}, len(fields)+1)
		colPtrs := make([]interface{}, len(cols))

		var grp sql.NullString
		colPtrs[0] = &grp

		avgVals := make([]sql.NullFloat64, len(fields))
		for i := range avgVals {
			colPtrs[i+1] = &avgVals[i]
		}

		if err := rows.Scan(colPtrs...); err != nil {
			return nil, err
		}

		item := &PageTimeItem{
			At:    "",
			Value: make(map[string]float64),
		}

		if grp.Valid {
			t, err := parseGroupTime(granularity, grp.String)
			if err != nil {
				return nil, fmt.Errorf("parse group time failed: %v", err)
			}
			item.At = strconv.FormatInt(t.Unix(), 10)
		} else {
			item.At = ""
		}

		for i, avgVal := range avgVals {
			if avgVal.Valid {
				item.Value[aggFieldNames[i]] = decimal.Round(avgVal.Float64, 4)
			} else {
				item.Value[aggFieldNames[i]] = 0
			}
		}

		result = append(result, item)
	}
	return result, nil
}

func (c *SQLiteCachePage[T]) GroupByFields(
	conditions map[string]any,
	groupFields []string,
	aggFields []AggField,
	sorts ...PageSorter,
) ([]*PageGroupItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(groupFields) == 0 {
		return nil, fmt.Errorf("groupFields cannot be empty")
	}
	if len(aggFields) == 0 {
		return nil, fmt.Errorf("aggFields cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	where, args := buildWhereClause(conditions)
	if where == "" {
		where = "WHERE 1=1"
	}

	groupExprs := make([]string, len(groupFields))
	groupAliases := make([]string, len(groupFields))
	for i, gf := range groupFields {
		alias := sanitizeColumnName(gf)
		groupAliases[i] = alias
		groupExprs[i] = fmt.Sprintf("json_extract(data, '$.%s') AS %s", gf, alias)
	}

	aggExprs := make([]string, len(aggFields))
	aggAliases := make([]string, len(aggFields))
	for i, af := range aggFields {
		alias := af.Alias
		if alias == "" {
			alias = sanitizeColumnName(af.Field)
		}
		aggAliases[i] = alias

		aggFunc := strings.ToUpper(string(af.Agg))
		if aggFunc == string(AggTotal) {
			aggFunc = string(AggSum)
		}

		aggExprs[i] = fmt.Sprintf("%s(CAST(json_extract(data, '$.%s') AS REAL)) AS %s", aggFunc, af.Field, alias)
	}

	selectFields := append(groupExprs, aggExprs...)
	groupBy := "GROUP BY " + strings.Join(groupAliases, ", ")
	orderBy := getOrderByClauseRaw(sorts)

	query := fmt.Sprintf(`SELECT %s FROM %s %s %s %s`, strings.Join(selectFields, ", "), c.table, where, groupBy, orderBy)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	result := make([]*PageGroupItem, 0)
	for rows.Next() {
		gCols := make([]sql.NullString, len(groupAliases))
		aCols := make([]sql.NullFloat64, len(aggAliases))
		scanArgs := make([]interface{}, 0, len(groupAliases)+len(aggAliases))
		for i := range gCols {
			scanArgs = append(scanArgs, &gCols[i])
		}
		for i := range aCols {
			scanArgs = append(scanArgs, &aCols[i])
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		item := &PageGroupItem{
			Group: make(map[string]string),
			Value: make(map[string]float64),
		}
		for i, gv := range gCols {
			if gv.Valid {
				item.Group[groupFields[i]] = gv.String
			}
		}
		for i, av := range aCols {
			if av.Valid {
				item.Value[aggAliases[i]] = decimal.Round(av.Float64, 4)
			} else {
				item.Value[aggAliases[i]] = 0
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (c *SQLiteCachePage[T]) Update(conditions map[string]any, value *T) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

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
		logger.Error(nil, fmt.Sprintf("Failed to update SQLite cache: %v", err))
	}
	return err
}

func (c *SQLiteCachePage[T]) Delete(conditions map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, cancel := context.WithTimeout(context.Background(), sqliteTimeOut)
	defer cancel()

	if len(conditions) == 0 {
		return fmt.Errorf("conditions cannot be empty")
	}

	where, args := buildWhereClause(conditions)
	query := fmt.Sprintf(`DELETE FROM %s %s`, c.table, where)

	_, err := c.db.Exec(query, args...)
	if err != nil {
		logger.Error(nil, fmt.Sprintf("Failed to delete from SQLite cache: %v", err))
	}
	return err
}

//func buildWhereClause(conditions map[string]any) (string, []any) {
//	if len(conditions) == 0 {
//		return "", nil
//	}
//	where := "WHERE "
//	args := make([]any, 0, len(conditions))
//	clauses := make([]string, 0, len(conditions))
//	for k, v := range conditions {
//		clauses = append(clauses, fmt.Sprintf("json_extract(data, '$.%s') = ?", k))
//		args = append(args, v)
//	}
//	where += strings.Join(clauses, " AND ")
//	return where, args
//}

func buildWhereClause(conditions map[string]any) (string, []any) {
	if len(conditions) == 0 {
		return "", nil
	}
	where := "WHERE "
	args := make([]any, 0)
	clauses := make([]string, 0)

	for k, v := range conditions {
		field := fmt.Sprintf("json_extract(data, '$.%s')", k)
		switch val := v.(type) {
		case map[string]any:
			for op, opVal := range val {
				var sqlOp string
				switch op {
				case "$lt":
					sqlOp = "<"
				case "$lte":
					sqlOp = "<="
				case "$gt":
					sqlOp = ">"
				case "$gte":
					sqlOp = ">="
				case "$ne":
					sqlOp = "!="
				case "$eq":
					sqlOp = "="
				case "$in":
					slice := toInterfaceSlice(opVal)
					if len(slice) == 0 {
						continue
					}
					placeholders := strings.Repeat("?,", len(slice))
					placeholders = strings.TrimSuffix(placeholders, ",")
					clauses = append(clauses, fmt.Sprintf("%s IN (%s)", field, placeholders))
					args = append(args, slice...)
					continue
				default:
					continue // unsupported
				}
				clauses = append(clauses, fmt.Sprintf("%s %s ?", field, sqlOp))
				args = append(args, opVal)
			}
		case []string:
			if len(val) == 0 {
				continue
			}
			placeholders := strings.Repeat("?,", len(val))
			placeholders = strings.TrimSuffix(placeholders, ",")
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", field, placeholders))
			for _, sv := range val {
				args = append(args, sv)
			}
		case []any:
			if len(val) == 0 {
				continue
			}
			placeholders := strings.Repeat("?,", len(val))
			placeholders = strings.TrimSuffix(placeholders, ",")
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", field, placeholders))
			args = append(args, val...)
		default:
			clauses = append(clauses, fmt.Sprintf("%s = ?", field))
			args = append(args, v)
		}
	}
	where += strings.Join(clauses, " AND ")
	return where, args
}

func toInterfaceSlice(v any) []any {
	switch arr := v.(type) {
	case []any:
		return arr
	case []string:
		res := make([]any, len(arr))
		for i, val := range arr {
			res[i] = val
		}
		return res
	case []int:
		res := make([]any, len(arr))
		for i, val := range arr {
			res[i] = val
		}
		return res
	case []int64:
		res := make([]any, len(arr))
		for i, val := range arr {
			res[i] = val
		}
		return res
	case []float64:
		res := make([]any, len(arr))
		for i, val := range arr {
			res[i] = val
		}
		return res
	default:
		return nil
	}
}

func parseGroupTime(granularity Granularity, grp string) (time.Time, error) {
	switch granularity {
	case GranularityMinute:
		return time.ParseInLocation("2006-01-02 15:04", grp, time.UTC)
	case Granularity5Minutes:
		return time.ParseInLocation("2006-01-02 15:04", grp, time.UTC)
	case Granularity10Minutes:
		return time.ParseInLocation("2006-01-02 15:04", grp, time.UTC)
	case Granularity30Minutes:
		return time.ParseInLocation("2006-01-02 15:04", grp, time.UTC)
	case GranularityHour:
		return time.ParseInLocation("2006-01-02 15", grp, time.UTC)
	case GranularityDay:
		return time.ParseInLocation("2006-01-02", grp, time.UTC)
	case GranularityMonth:
		return time.ParseInLocation("2006-01", grp, time.UTC)
	case GranularityYear:
		return time.ParseInLocation("2006", grp, time.UTC)
	case GranularityWeek:
		parts := strings.Split(grp, "-")
		if len(parts) != 2 {
			return time.Time{}, fmt.Errorf("invalid week format: %s", grp)
		}
		year, err1 := strconv.Atoi(parts[0])
		week, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || week < 1 || week > 53 {
			return time.Time{}, fmt.Errorf("invalid year or week: %s", grp)
		}
		return getWeekStartTime(year, week), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported granularity: %s", granularity)
	}
}

func getWeekStartTime(year, week int) time.Time {
	t := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, 1)
	}
	return t.AddDate(0, 0, (week-1)*7)
}

func getOrderByClauseRaw(sorts []PageSorter) string {
	if len(sorts) == 0 {
		return ""
	}
	orderClauses := make([]string, len(sorts))
	for i, sort := range sorts {
		dir := "DESC"
		if sort.Asc {
			dir = "ASC"
		}
		orderClauses[i] = fmt.Sprintf("%s %s", sort.SortField, dir)
	}
	return "ORDER BY " + strings.Join(orderClauses, ", ")
}

func sanitizeColumnName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "`", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "'", "")
	return strings.ReplaceAll(name, ".", "_")
}
