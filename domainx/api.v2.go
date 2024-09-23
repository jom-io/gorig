package domainx

import (
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/utils/errors"
)

//func UseConnect(conType ConType, dbName string, table string) *Con {
//	con := UseCon(conType, dbName, table)
//	return &Connect{Con: con}
//}

// GetByID 获取单条记录
func (c *Con) GetByID(id int64) (err *errors.Error) {
	return GetByID(c, id, c)
}

//// MustGetByID 获取单条记录
//func (c *Con) MustGetByID(id int64) *errors.Error {
//	return MustGetByID(c, id, )
//}

// FindByIDs 根据ID列表查询多条记录
func (c *Con) FindByIDs(ids []int64, result *[]Identifiable) *errors.Error {
	return FindByIDs(c, ids, result)
}

// SaveOrUpdate 新增或者根据id更新
func (c *Con) SaveOrUpdate(data Identifiable, newIDs ...int64) (id int64, err *errors.Error) {
	return SaveOrUpdate(c, data, newIDs...)
}

// Delete 删除
func (c *Con) Delete(id int64) *errors.Error {
	return Delete(c, id)
}

// GetByField 根据字段名称和值查询单条记录
func (c *Con) GetByField(fieldName string, value interface{}, result *Identifiable) (err *errors.Error) {
	return GetByField(c, fieldName, value, result)
}

// GetByCondition 根据条件查询单条记录
func (c *Con) GetByCondition(condition map[string]interface{}, result *Identifiable) (err *errors.Error) {
	return GetByCondition(c, condition, result)
}

// FindByField 根据字段名称和值查询多条记录 最多返回1000条
func (c *Con) FindByField(fieldName string, value interface{}, result *[]Identifiable, prefixes ...string) (err *errors.Error) {
	return FindByField(c, fieldName, value, result, prefixes...)
}

// FindByCondition 根据条件查询多条记录 最多返回1000条
func (c *Con) FindByCondition(condition map[string]interface{}, result *[]Identifiable, prefixes ...string) (err *errors.Error) {
	return FindByCondition(c, condition, result, prefixes...)
}

// FindByMatch 根据条件查询多条记录 最多返回1000条
func (c *Con) FindByMatch(matchList []Match, result *[]Identifiable, prefixes ...string) (err *errors.Error) {
	return FindByMatch(c, matchList, result, prefixes...)
}

// GetByMatch 根据条件查询单条记录
func (c *Con) GetByMatch(matchList []Match, result *Identifiable) (err *errors.Error) {
	return GetByMatch(c, matchList, result)
}

// CountByField 根据字段名称和值查询数量
func (c *Con) CountByField(fieldName string, value interface{}) (int64, *errors.Error) {
	return CountByField(c, fieldName, value)
}

// CountByCondition 根据条件查询数量
func (c *Con) CountByCondition(condition map[string]interface{}) (int64, *errors.Error) {
	return CountByCondition(c, condition)
}

// CountByMatch 根据条件查询数量
func (c *Con) CountByMatch(matchList []Match) (int64, *errors.Error) {
	return CountByMatch(c, matchList)
}

// UpdatePart 根据ID更新部分字段
func (c *Con) UpdatePart(id int64, data map[string]interface{}) *errors.Error {
	return UpdatePart(c, id, data)
}

// FindByPageField FiledName查询分页
func (c *Complex[T]) FindByPageField(fieldName string, value interface{}, page *load.Page, pageResp *load.PageResp, prefixes ...string) *errors.Error {
	return FindByPageField[*Complex[T]](c.Con, fieldName, value, page, pageResp, nil, prefixes...)
}

// FindByPage Condition查询分页
func (c *Complex[T]) FindByPage(condition map[string]interface{}, page *load.Page, pageResp *load.PageResp, prefixes ...string) *errors.Error {
	return FindByPage[*Complex[T]](c.Con, condition, page, pageResp, nil, prefixes...)
}

func (c *Complex[T]) FindByPageMatch(matchList []Match, page *load.Page, pageResp *load.PageResp, prefixes ...string) *errors.Error {
	return FindByPageMatch[*Complex[T]](c.Con, matchList, page, pageResp, nil, prefixes...)
}
