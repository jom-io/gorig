package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jom-io/gorig/utils/logger"
	"path/filepath"
)

type PageStorage[T any] interface {
	Put(value T) error
	Find(page, size int64, conditions map[string]any, sort ...PageSorter) (*PageCache[T], error)
	Get(conditions map[string]any) (*T, error)
	Count(conditions map[string]any) (int64, error)
	Update(conditions map[string]any, value *T) error
}

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

func NewPageStorage[T any](ctx context.Context, t Type, args ...any) PageStorage[T] {
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
