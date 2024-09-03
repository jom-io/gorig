package load

import (
	"github.com/gin-gonic/gin"
	"gorig/utils/logger"
)

type Page struct {
	Page   int64 `json:"page" form:"page"`
	Size   int64 `json:"size" form:"size"`
	LastID int64 `json:"lastID" form:"lastID"`
}

type PageResp struct {
	Page   int64       `json:"page"`
	Size   int64       `json:"size"`
	Total  int64       `json:"total"`
	LastID int64       `json:"lastID"`
	Result interface{} `json:"result"`
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

type Identifiable interface {
	GetID() int64
}

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

func (r *PageResp) Build(page *Page, total int64, LastID int64, result interface{}) {
	r.Page = page.Page
	r.Size = page.Size
	r.Total = total
	r.LastID = LastID
	r.Result = result
}

func (r *PageResp) BuildS(page *Page, LastID int64, result interface{}) {
	r.Page = page.Page
	r.Size = page.Size
	r.LastID = LastID
	r.Result = result
}

func GetLastID[T Identifiable](conList []T) int64 {
	if len(conList) > 0 {
		return conList[len(conList)-1].GetID()
	}
	return 0
}
