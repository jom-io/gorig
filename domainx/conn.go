package domainx

import (
	"github.com/jom-io/gorig/utils/errors"
	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/field"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

type ConType string

const (
	Mysql ConType = "mysql"
	Redis ConType = "redis"
	Mongo ConType = "mongo"
)

func (c ConType) String() string {
	return string(c)
}

type Con struct {
	ID       int64 `gorm:"primaryKey" bson:"id" json:"id"`
	ConType  `gorm:"-" bson:"-" json:"-"`
	*gorm.DB `gorm:"-" bson:"-" json:"-"`
	MDB      *qmgo.Client `gorm:"-" bson:"-" json:"-"`
	DBName   string       `gorm:"-" bson:"-" json:"-"`
	gTabel   string       `gorm:"-" bson:"-" json:"-"`
	gSort    *[]*Sort     `gorm:"-" bson:"-" json:"-"`
}

func (c *Con) GetConType() ConType {
	// 默认mysql
	if c.ConType == "" {
		return Mysql
	}
	return c.ConType
}

func (c *Con) GenerateId() int64 {
	// 创建一个新的随机数生成器，使用当前时间戳作为种子
	randSource := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(randSource)
	// 获取当前时间戳的毫秒表示形式
	currentTimestampMillis := time.Now().UnixNano() / 1_000_000
	// 生成3位随机数
	randomNumber := rnd.Intn(900) + 100
	// 将当前时间戳和3位随机数组合起来
	result := currentTimestampMillis*1000 + int64(randomNumber)
	return result
}

func (c *Con) GenerateSetID() int64 {
	c.ID = c.GenerateId()
	return c.ID
}

func (c *Con) MustGetDB() (any, *errors.Error) {
	db := c.GetDB()
	if db == nil {
		return nil, errors.Sys("数据库连接失败")
	}
	return db, nil
}

type Result[T any] struct {
	Data T
}

type ConTable interface {
	//GetTableName() string
	TableName() string
	GetCon() *Con
}

func (c *Con) TableName() string {
	if c == nil {
		return ""
	}
	if c.gTabel != "" {
		return c.gTabel
	}
	if c.DB != nil {
		return c.DB.Statement.Table
	}
	return ""
}

func (c *Con) AddSort(field string, asc bool, prefix ...string) *Con {
	if c.gSort == nil {
		c.gSort = new([]*Sort)
	}
	pre := ""
	if len(prefix) > 0 {
		for _, v := range prefix {
			pre += v + "."
		}
	}
	*c.gSort = append(*c.gSort, &Sort{Field: field, Asc: asc, Prefix: pre})
	return c
}

func (c *Con) SetSort(s ...*Sort) *Con {
	c.gSort = &s
	return c
}

func (c *Con) GetSort() *[]*Sort {
	return c.gSort
}

//func (c *Con) GetTableName() string {
//	if c == nil {
//		return ""
//	}
//	return c.TableName
//}

func (c *Con) GetCon() *Con {
	return c
}

// CustomFields 指定自定义field的field名
func (c *Options) CustomFields() field.CustomFieldsBuilder {
	return field.NewCustom().SetCreateAt("created_at").SetUpdateAt("updated_at")
}

func (c *Con) GetID() int64 {
	return c.ID
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

type Options struct {
	CreatedAt time.Time      `gorm:"autoCreateTime:second;" bson:"createAt" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime:second" bson:"updateAt" json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" bson:"-"`
}

type IdxType string

const (
	Unique IdxType = "unique"
	Idx    IdxType = "idx"
)

type Index struct {
	IdxType IdxType
	Fields  []string
	IdxName string
}

var migrationList []*migration

type migration struct {
	DBFunc func() ConTable
	Index  []Index
}

type IndexFunc func() []Index

type MatchType string

const (
	MEq    MatchType = "="
	MEqr   MatchType = "="
	MLt    MatchType = "<"
	MLte   MatchType = "<="
	MGt    MatchType = ">"
	MGte   MatchType = ">="
	MNE    MatchType = "!="
	MLIKE  MatchType = "like"
	MIN    MatchType = "in"
	MNOTIN MatchType = "not in"
)

type Match struct {
	Field string
	Value interface{}
	Type  MatchType
}

type Matches []Match

func (m Matches) Add(field string, value interface{}, t MatchType, ignore ...bool) Matches {
	if value == nil {
		return m
	}
	if len(ignore) == 0 || !ignore[0] {
		// 根据类型判断是否取值 字符串判断非"" 数字判断非0 数组判断长度大于0 map判断长度大于0 struct判断是否有值
		switch value.(type) {
		case string:
			if value.(string) == "" {
				return m
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			if cast.ToInt64(value) == 0 {
				return m
			}
		case float32, float64:
			if value.(float64) == 0 {
				return m
			}
		case []interface{}:
			if len(value.([]interface{})) == 0 {
				return m
			}
		case map[string]interface{}:
			if len(value.(map[string]interface{})) == 0 {
				return m
			}
		case struct{}:
			if value == struct{}{} {
				return m
			}
		}
	}
	return append(m, Match{Field: field, Value: value, Type: t})
}

func (m Matches) Like(field string, value string, ignore ...bool) Matches {
	return m.Add(field, value, MLIKE, ignore...)
}

func (m Matches) Eq(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MEq, ignore...)
}

func (m Matches) Lt(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MLt, ignore...)
}

func (m Matches) Lte(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MLte, ignore...)
}

func (m Matches) Gt(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MGt, ignore...)
}

func (m Matches) Gte(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MGte, ignore...)
}

func (m Matches) Ne(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MNE, ignore...)
}

func (m Matches) In(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MIN, ignore...)
}

func (m Matches) NotIn(field string, value interface{}, ignore ...bool) Matches {
	return m.Add(field, value, MNOTIN, ignore...)
}

func (m Matches) AddMatch(match Match) Matches {
	return append(m, match)
}

func (m Matches) AddMatches(matches Matches) Matches {
	return append(m, matches...)
}

type Sort struct {
	Field  string
	Asc    bool
	Prefix string
}

type Complex[T any] struct {
	*Con
	Data *T `bson:"data" gorm:"embedded"`
	Options
}

func (u *Complex[T]) CustomFields() field.CustomFieldsBuilder {
	return field.NewCustom().SetCreateAt("createAt").SetUpdateAt("updateAt")
}

func (c *Complex[T]) IsNil() bool {
	return c == nil || c.Con == nil || c.ID == 0
}
