package domainx

import (
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/utils/errors"
)

func (c *Con) FindByIDs(ids []int64, result *[]Identifiable) *errors.Error {
	return FindByIDs(c, ids, result)
}

func (c *Con) SaveOrUpdate(data Identifiable, newIDs ...int64) (id int64, err *errors.Error) {
	return SaveOrUpdate(c, data, newIDs...)
}

func (c *Con) Save(data Identifiable, newIDs ...int64) (id int64, err *errors.Error) {
	return Save(c, data, newIDs...)
}

func (c *Con) Delete(data Identifiable) *errors.Error {
	return Delete(c, data)
}

func (c *Con) GetByField(fieldName string, value interface{}, result Identifiable) (err *errors.Error) {
	return GetByField(c, fieldName, value, &result)
}

func (c *Con) GetByCondition(condition map[string]interface{}, result *Identifiable) (err *errors.Error) {
	return GetByCondition(c, condition, result)
}

func (c *Con) FindByField(fieldName string, value interface{}, result *[]Identifiable, prefixes ...string) (err *errors.Error) {
	return FindByField(c, fieldName, value, result, prefixes...)
}

func (c *Con) FindByCondition(condition map[string]interface{}, result *[]Identifiable, prefixes ...string) (err *errors.Error) {
	return FindByCondition(c, condition, result, prefixes...)
}

func (c *Con) FindByMatch(matchList []Match, result *[]Identifiable, prefixes ...string) (err *errors.Error) {
	return FindByMatch(c, matchList, result, prefixes...)
}

func (c *Con) GetByMatch(matchList []Match, result *Identifiable) (err *errors.Error) {
	return GetByMatch(c, matchList, result)
}

func (c *Con) CountByField(fieldName string, value interface{}) (int64, *errors.Error) {
	return CountByField(c, fieldName, value)
}

func (c *Con) CountByCondition(condition map[string]interface{}) (int64, *errors.Error) {
	return CountByCondition(c, condition)
}

func (c *Con) CountByMatch(matchList []Match) (int64, *errors.Error) {
	return CountByMatch(c, matchList)
}

func (c *Con) UpdatePart(id int64, data map[string]interface{}) *errors.Error {
	return UpdatePart(c, id, data)
}

func (c *Complex[T]) FindByPageField(fieldName string, value interface{}, page *load.Page, pageResp *load.PageResp, prefixes ...string) *errors.Error {
	return FindByPageField[*Complex[T]](c.Con, fieldName, value, page, pageResp, nil, prefixes...)
}

func (c *Complex[T]) FindByPage(condition map[string]interface{}, page *load.Page, pageResp *load.PageResp, prefixes ...string) *errors.Error {
	return FindByPage[*Complex[T]](c.Con, condition, page, pageResp, nil, prefixes...)
}

func (c *Complex[T]) FindByPageMatch(matchList []Match, page *load.Page, pageResp *load.PageResp, prefixes ...string) *errors.Error {
	return FindByPageMatch[*Complex[T]](c.Con, matchList, page, pageResp, nil, prefixes...)
}
