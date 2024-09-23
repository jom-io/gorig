package load

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
)

type Page struct {
	Page   int64 `json:"page" form:"page"`
	Size   int64 `json:"size" form:"size"`
	LastID int64 `json:"lastID" form:"lastID"`
}

type PageResp struct {
	Page   int64  `json:"page"`
	Size   int64  `json:"size"`
	Total  *Total `json:"total"`
	LastID int64  `json:"lastID"`
	Result any    `json:"result"`
}

type PageRespT[T any] struct {
	Page   int64  `json:"page"`
	Size   int64  `json:"size"`
	Total  *Total `json:"total"`
	LastID int64  `json:"lastID"`
	Result *[]T   `json:"result"`
}

type Total int64

func (t *Total) Set(total int64) {
	if t == nil {
		t = new(Total)
	}
	*t = Total(total)
}

func (p *Page) Offset() int64 {
	return (p.Page - 1) * p.Size
}

func (p *Page) Limit() int64 {
	return p.Size
}

func (p *Page) NextPage() int64 {
	return p.Page + 1
}

func (p *Page) PrevPage() int64 {
	return p.Page - 1
}

func (p *Page) SetPage(page int64) {
	p.Page = page
}

//type Identifiable interface {
//	GetID() int64
//}

func BuildPage(ctx *gin.Context, page, pageSize, lastId int64) *Page {
	if page <= 0 {
		logger.Warn(ctx, "page is less than 0, set to 1")
		page = 1
	}
	if pageSize <= 0 {
		logger.Warn(ctx, "pageSize is less than 0, set to 10")
		pageSize = 10
	}
	if pageSize > 10000 {
		logger.Warn(ctx, "pageSize is too large, set to 10000")
		pageSize = 10000
	}
	if lastId < 0 {
		logger.Warn(ctx, "lastId is less than 0, set to 0")
		lastId = 0
	}
	return &Page{
		Page:   page,
		Size:   pageSize,
		LastID: lastId,
	}
}

func (r *PageResp) Build(page *Page, total *Total, LastID int64, result any) {
	r.Page = page.Page
	r.Size = page.Size
	r.Total = total
	r.LastID = LastID
	r.Result = result
}

func (r *PageResp) BuildS(page *Page, LastID int64, result any) {
	r.Page = page.Page
	r.Size = page.Size
	r.LastID = LastID
	r.Result = result
}

func Covert[T any](r *PageResp) *PageRespT[T] {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(nil, "CovertT", zap.Any("err", err))
		}
	}()
	if r == nil {
		return &PageRespT[T]{
			Result: &[]T{},
		}
	}
	result := new([]T)
	// 改为用JSON序列化反序列化
	if r.Result != nil {
		b, _ := json.Marshal(r.Result)
		if e := json.Unmarshal(b, result); e != nil {
			logger.Error(nil, "CovertT", zap.Any("err", e))
		} else {
			return &PageRespT[T]{
				Page:   r.Page,
				Size:   r.Size,
				Total:  r.Total,
				LastID: r.LastID,
				Result: result,
			}
		}
	}
	return &PageRespT[T]{
		Result: &[]T{},
	}
}

func (t *PageRespT[T]) ParsePageResp(r *PageResp, result *[]T) *PageRespT[T] {
	if t == nil {
		t = &PageRespT[T]{}
	}
	t.Page = r.Page
	t.Size = r.Size
	t.Total = r.Total
	t.LastID = r.LastID
	t.Result = result
	return t
}
