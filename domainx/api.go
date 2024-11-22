package domainx

import (
	"fmt"
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"strings"
)

func UseCon(conType ConType, dbName string, table string) *Con {
	con := new(Con)
	//con.ID = new(ID)
	con.ConType = conType
	con.DBName = dbName
	con.GTable = table
	switch conType {
	case Mysql:
		if connDb := UseDbConn(dbName); connDb != nil {
			con.DB = connDb
			return con
		}
	case Redis:
	case Mongo:
		if coneDb := UseMongoDbConn(dbName); coneDb != nil {
			con.MDB = coneDb
			return con
		}
	}
	return nil
}

func CtIdx(idxType IdxType, fileds ...string) Index {
	// 校验不能存在空字符串和重复字段和-
	for _, v := range fileds {
		if v == "" {
			logger.Logger.Fatal("CreateIdx field is nil")
		}
		if strings.Contains(v, "-") {
			logger.Logger.Fatal("CreateIdx field can not contain -")
		}
	}
	// 校验重复字段
	for i := 0; i < len(fileds); i++ {
		for j := i + 1; j < len(fileds); j++ {
			if fileds[i] == fileds[j] {
				logger.Logger.Fatal("CreateIdx field can not repeat")
			}
		}
	}
	var g Index
	g.IdxType = idxType
	g.Fields = fileds
	g.IdxName = string(idxType) + "-" + strings.Join(fileds, "-")
	return g
}

func AutoMigrate(getDB func() (value ConTable), index ...Index) {
	mInfo := &Migration{
		DBFunc: getDB,
		Index:  index,
	}
	MigrationList = append(MigrationList, mInfo)
}

func (c *Con) HandleWithErr(err error) (error *errors.Error) {
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil
		}
		error = errors.Sys(fmt.Sprintf("%s数据库操作失败: %s", c.TableName(), err.Error()))
		return error
	}
	return nil
}

func unknownDBType() *errors.Error {
	return errors.Sys("未知的数据库类型")
}

// GetByID 获取单条记录
func GetByID[T any](c *Con, id int64, result *T) (err *errors.Error) {
	if id <= 0 {
		return nil
	}
	if c == nil {
		return errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.GetByID(c, id, result)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// MustGetByID 获取单条记录
func MustGetByID[T Identifiable](c *Con, id int64, result T) *errors.Error {
	err := GetByID(c, id, &result)
	if err != nil {
		return err
	}
	if &result == nil || result.GetID().IsNil() {
		return errors.Verify("未找到该记录")
	}
	return nil
}

// FindByIDs
func FindByIDs[T any](c *Con, ids []int64, result *[]T) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}

	matchList := new(Matches).In("id", ids)
	return FindByMatch(c, *matchList, result, "con")
}

// SaveOrUpdate 新增或者根据id更新
func SaveOrUpdate(c *Con, data Identifiable, newIDs ...int64) (id int64, err *errors.Error) {
	if c == nil {
		return 0, errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())
	if !data.GetID().IsNil() {
		c.ID = data.GetID().Int64()
	}

	newID := int64(0)
	if len(newIDs) > 0 {
		newID = newIDs[0]
	}
	id, gErr := dbService.Save(c, data, newID)
	if gErr != nil {
		return 0, c.HandleWithErr(gErr)
	}

	return id, nil
}

// Delete 删除
func Delete(c *Con, data Identifiable) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.Delete(c, data)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// toSnake 驼峰转蛇形
func toSnake(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return string(data)
}

// GetByField 根据字段名称和值查询单条记录
func GetByField[T any](c *Con, fieldName string, value interface{}, result *T) (err *errors.Error) {
	// 将condition转为matchList 使用=连接
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return GetByMatch(c, matchList, result)
}

// GetByCondition 根据条件查询单条记录
func GetByCondition[T any](c *Con, condition map[string]interface{}, result *T) (err *errors.Error) {
	// 将condition转为matchList 使用=连接
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return GetByMatch(c, matchList, result)

}

// FindByField 根据字段名称和值查询多条记录 最多返回1000条
func FindByField[T any](c *Con, fieldName string, value interface{}, result *[]T, prefixes ...string) (err *errors.Error) {
	// 将condition转为matchList 使用=连接
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return FindByMatch(c, matchList, result, prefixes...)
}

// FindByCondition 根据条件查询多条记录 最多返回1000条
func FindByCondition[T any](c *Con, condition map[string]interface{}, result *[]T, prefixes ...string) (err *errors.Error) {
	// 将condition转为matchList 使用=连接
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return FindByMatch(c, matchList, result, prefixes...)
}

// FindByMatch 根据条件查询多条记录 最多返回1000条
func FindByMatch[T any](c *Con, matchList []Match, result *[]T, prefixes ...string) (err *errors.Error) {
	if c == nil {
		return errors.Sys("con not init")
	}
	if result == nil {
		return errors.Sys("result is nil")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.FindByMatch(c, matchList, result, prefixes...)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// GetByMatch 根据条件查询单条记录
func GetByMatch[T any](c *Con, matchList []Match, result *T) (err *errors.Error) {
	if c == nil {
		return errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.GetByMatch(c, matchList, result)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// CountByField 根据字段名称和值查询数量
func CountByField(c *Con, fieldName string, value interface{}) (int64, *errors.Error) {
	// 将condition转为matchList 使用=连接
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return CountByMatch(c, matchList)
}

// CountByCondition 根据条件查询数量
func CountByCondition(c *Con, condition map[string]interface{}) (int64, *errors.Error) {
	// 将condition转为matchList 使用=连接
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return CountByMatch(c, matchList)
}

// CountByMatch 根据条件查询数量
func CountByMatch(c *Con, matchList []Match) (int64, *errors.Error) {
	if c == nil {
		return 0, errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	count, gErr := dbService.CountByMatch(c, matchList)
	if gErr != nil {
		return 0, c.HandleWithErr(gErr)
	}
	return count, nil
}

// UpdatePart 根据ID更新部分字段
func UpdatePart(c *Con, id int64, data map[string]interface{}) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.UpdatePart(c, id, data)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// FindByPageField FiledName查询分页
func FindByPageField[T Identifiable](c *Con, fieldName string, value interface{}, page *load.Page, pageResp *load.PageResp, result *[]T, prefixes ...string) *errors.Error {
	// 将condition转为matchList 使用=连接
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return FindByPageMatch(c, matchList, page, pageResp, result, prefixes...)
}

// FindByPage Condition查询分页
func FindByPage[T Identifiable](c *Con, condition map[string]interface{}, page *load.Page, pageResp *load.PageResp, result *[]T, prefixes ...string) *errors.Error {
	// 将condition转为matchList 使用=连接
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return FindByPageMatch(c, matchList, page, pageResp, result, prefixes...)
}

func FindByPageMatch[T Identifiable](c *Con, matchList []Match, page *load.Page, pageResp *load.PageResp, result *[]T, prefixes ...string) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}
	if result == nil {
		result = &[]T{}
	}
	if pageResp == nil {
		return errors.Sys("pageResp is nil")
	}

	dbService := GetDBService(c.GetConType())

	total := new(load.Total)
	gErr := dbService.FindByPageMatch(c, matchList, page, total, result, prefixes...)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	pageResp.Build(page, total, GetLastID(*result), result)
	return nil
}
