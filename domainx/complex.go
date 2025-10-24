package domainx

import (
	"context"
	"github.com/jom-io/gorig/global/variable"
	"gorm.io/gorm"
	"time"
)

type Options struct {
	CreatedAt time.Time      `gorm:"autoCreateTime:second;" bson:"createAt" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime:second" bson:"updateAt" json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" bson:"-"`
}

func (o *Options) SaveCreate() {
	o.CreatedAt = time.Now()
}

func (o *Options) SaveUpdate() {
	o.UpdatedAt = time.Now()
}

type Complex[T any] struct {
	*Con
	Data *T `bson:"data" gorm:"embedded" json:"data"`
	Options
}

func NewComplex[T any](conType ConType, dbName string, table string) *Complex[T] {
	return UseComplex[T](conType, dbName, table)
}

func (c *Complex[T]) TableName() string {
	if c.Con != nil {
		return c.Con.TableName()
	}
	return ""
}

func (c *Complex[T]) GetID() ID {
	if c == nil || c.Con == nil {
		return ID(0)
	}
	return c.Con.GetID()
}

func (c *Complex[T]) IsNil() bool {
	return c == nil || c.Con == nil || c.GetID().IsNil()
}

func UseComplex[T any](conType ConType, dbName string, table string, prefix ...string) *Complex[T] {
	return CreateComplex[T](context.Background(), conType, dbName, table, nil, prefix...)
}

func UseComplexD[T any](conType ConType, dbName string, table string) Complex[T] {
	return *UseComplex[T](conType, dbName, table)
}

func CreateComplex[T any](ctx context.Context, conType ConType, dbName string, table string, data *T, prefix ...string) *Complex[T] {
	if len(prefix) > 0 {
		for i := range prefix {
			table = prefix[i] + table
		}
	}

	var newData T
	if any(data) != nil && data != nil {
		newData = *data
	}

	c := Complex[T]{Con: UseCon(ctx, conType, dbName, table), Data: &newData}
	if c.Con == nil {
		return &c
	}

	if variable.TBPrefix != "" {
		c.Con.GTable = variable.TBPrefix + c.Con.GTable
	}

	c.Con.SaveCreateTime = func() {
		c.SaveCreate()
	}
	c.Con.SaveUpdateTime = func() {
		c.SaveUpdate()
	}
	return &c
}

type ComplexList[T any] []*Complex[T]

func (c *ComplexList[T]) List() []*T {
	if c == nil {
		return nil
	}
	respList := make([]*T, 0)
	for _, v := range *c {
		if v != nil && v.Data != nil {
			respList = append(respList, v.Data)
		}
	}
	return respList
}
