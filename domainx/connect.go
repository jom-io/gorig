package domainx

import (
	"github.com/jom-io/gorig/utils/errors"
	"github.com/qiniu/qmgo"
	"gorm.io/gorm"
)

type Con struct {
	ID             int64 `gorm:"primaryKey" bson:"id" json:"id"`
	ConType        `gorm:"-" bson:"-" json:"-"`
	*gorm.DB       `gorm:"-" bson:"-" json:"-"`
	MDB            *qmgo.Client `gorm:"-" bson:"-" json:"-"`
	DBName         string       `gorm:"-" bson:"-" json:"-"`
	GTable         string       `gorm:"-" bson:"-" json:"-"`
	Sort           Sorts        `gorm:"-" bson:"-" json:"-"`
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
	if c.DB != nil {
		return c.DB.Statement.Table
	}
	return ""
}

func (c *Con) GetCon() *Con {
	return c
}

func (c *Con) GetDB() any {
	switch c.ConType {
	case Mysql:
		return c.DB
	case Mongo:
		return c.MDB
	default:
		return c.DB
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
