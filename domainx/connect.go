package domainx

import (
	"context"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/qiniu/qmgo"
	"gorm.io/gorm"
	"strings"
)

type Con struct {
	Ctx            context.Context `gorm:"-" bson:"-" json:"-"`
	ID             int64           `gorm:"primaryKey" bson:"id" json:"id"`
	ConType        `gorm:"-" bson:"-" json:"-"`
	MysqlDB        *gorm.DB     `gorm:"-" bson:"-" json:"-"`
	MongoDB        *qmgo.Client `gorm:"-" bson:"-" json:"-"`
	DBName         string       `gorm:"-" bson:"-" json:"-"`
	GTable         string       `gorm:"-" bson:"-" json:"-"`
	Sort           Sorts        `gorm:"-" bson:"-" json:"-"`
	SelectFields   []string     `gorm:"-" bson:"-" json:"-"`
	OmitFields     []string     `gorm:"-" bson:"-" json:"-"`
	SaveCreateTime func()       `gorm:"-" bson:"-" json:"-"`
	SaveUpdateTime func()       `gorm:"-" bson:"-" json:"-"`
}

//type Connect struct {
//	*Con `bson:",inline" gorm:"embedded"`
//}

type ConTable interface {
	TableName() string
	GetCon() *Con
}

type Identifiable interface {
	GetID() ID
	SetID(id int64)
}

func GetLastID[T Identifiable](conList []T) int64 {
	if len(conList) > 0 {
		return conList[len(conList)-1].GetID().Int64()
	}
	return 0
}

func (c *Con) MustGetDB() (any, *errors.Error) {
	db := c.GetDB()
	if db == nil {
		return nil, errors.Sys("DB connection failed")
	}
	return db, nil
}

type Result[T any] struct {
	Data T
}

func (c *Con) TableName() string {
	if c == nil {
		return ""
	}
	if c.GTable != "" {
		return c.GTable
	}
	if c.MysqlDB != nil {
		return c.MysqlDB.Statement.Table
	}
	return ""
}

func (c *Con) GetCon() *Con {
	return c
}

func (c *Con) GetDB() any {
	switch c.ConType {
	case Mysql:
		return c.MysqlDB
	case Mongo:
		return c.MongoDB
	default:
		return c.MysqlDB
	}
}

func (c *Con) GetID() ID {
	if c == nil {
		return 0
	}
	return ID(c.ID)
}

func (c *Con) SetID(id int64) {
	c.ID = id
}

func (c *Con) GenerateId() int64 {
	return c.GenerateID()
}

func (c *Con) GenerateID() int64 {
	return c.GetID().GenerateID()
}

func (c *Con) GenerateSetID() {
	c.ID = c.GenerateID()
}

func (c *Con) SetSelectFields(fields ...string) {
	c.SelectFields = sanitizeFields(fields...)
}

func (c *Con) SetOmitFields(fields ...string) {
	c.OmitFields = sanitizeFields(fields...)
}

func sanitizeFields(fields ...string) []string {
	if len(fields) == 0 {
		return nil
	}
	cleaned := make([]string, 0, len(fields))
	seen := make(map[string]struct{})
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		cleaned = append(cleaned, f)
	}
	return cleaned
}
