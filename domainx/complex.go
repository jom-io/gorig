package domainx

import (
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

func (c *Complex[T]) GetID() *ID {
	if c == nil {
		return new(ID)
	}
	return c.Con.GetID()
}

func (c *Complex[T]) IsNil() bool {
	return c == nil || c.Con == nil || c.GetID().IsNil()
}

func UseComplex[T any](conType ConType, dbName string, table string) *Complex[T] {
	c := Complex[T]{Con: UseCon(conType, dbName, table)}
	c.Con.SaveCreateTime = func() {
		c.SaveCreate()
	}
	c.Con.SaveUpdateTime = func() {
		c.SaveUpdate()
	}
	return &c
}

func UseComplexD[T any](conType ConType, dbName string, table string) Complex[T] {
	return *UseComplex[T](conType, dbName, table)
}
