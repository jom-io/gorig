package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jom-io/gorig/utils/logger"
	"path/filepath"
	"time"
)

type Pager[T any] interface {
	Put(value T) error
	Find(page, size int64, conditions map[string]any, sort ...PageSorter) (*PageCache[T], error)
	Get(conditions map[string]any) (*T, error)
	Count(conditions map[string]any) (int64, error)
	Update(conditions map[string]any, value *T) error
	Delete(conditions map[string]any) error
	GroupByTime(conditions map[string]any, from, to time.Time, granularity Granularity, agg Agg, fields ...string) ([]*PageTimeItem, error)
}

type Granularity string

const (
	GranularityMinute    Granularity = "minute"
	GranularityHour      Granularity = "hour"
	GranularityDay       Granularity = "day"
	GranularityWeek      Granularity = "week"
	GranularityMonth     Granularity = "month"
	GranularityYear      Granularity = "year"
	Granularity5Minutes  Granularity = "5minutes"
	Granularity10Minutes Granularity = "10minutes"
	Granularity30Minutes Granularity = "30minutes"
)

type Agg string

const (
	AggSum   Agg = "sum"
	AggAvg   Agg = "avg"
	AggMax   Agg = "max"
	AggMin   Agg = "min"
	AggCount Agg = "count"
	AggTotal Agg = "total"
)

type PageSorter struct {
	SortField string
	Asc       bool
}

type PageCache[T any] struct {
	Total int64 `json:"total"`
	Page  int64 `json:"page"`
	Size  int64 `json:"size"`
	Items []*T  `json:"items"`
}

type PageTimeItem struct {
	At    string             `json:"at"`
	Value map[string]float64 `json:"value"`
}

func (p *PageCache[T]) JSON() string {
	jsonStr, err := json.Marshal(p)
	if err != nil {
		return "{}"
	}
	return string(jsonStr)
}

func PageSorterAsc(field string) PageSorter {
	return PageSorter{
		SortField: field,
		Asc:       true,
	}
}

func PageSorterDesc(field string) PageSorter {
	return PageSorter{
		SortField: field,
		Asc:       false,
	}
}

func NewPager[T any](ctx context.Context, t Type, args ...any) Pager[T] {
	switch t {
	case Sqlite:
		if len(args) < 1 {
			args = append(args, filepath.Base(fmt.Sprintf("%T", new(T))))
		}
		cache, err := NewSQLiteCachePage[T](args[0].(string))
		if err != nil {
			logger.Error(ctx, fmt.Sprintf("Failed to create SQLite cache: %v", err))
		}
		return cache
	default:
		logger.Error(ctx, fmt.Sprintf("Unsupported cache type: %s, using memory cache", t))
		return nil
	}
}
