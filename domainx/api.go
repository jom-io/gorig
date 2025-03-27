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
	// Validate that there are no empty strings, duplicate fields, or hyphens
	for _, v := range fileds {
		if v == "" {
			logger.Logger.Fatal("CreateIdx field is nil")
		}
		if strings.Contains(v, "-") {
			logger.Logger.Fatal("CreateIdx field can not contain -")
		}
	}
	// Validate duplicate fields
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
		error = errors.Sys(fmt.Sprintf("%s database operation failed: %s", c.TableName(), err.Error()))
		return error
	}
	return nil
}

func unknownDBType() *errors.Error {
	return errors.Sys("Unknown database type")
}

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

func MustGetByID[T Identifiable](c *Con, id int64, result T) *errors.Error {
	err := GetByID(c, id, &result)
	if err != nil {
		return err
	}
	if &result == nil || result.GetID().IsNil() {
		return errors.Verify("Record not found")
	}
	return nil
}

func FindByIDs[T any](c *Con, ids []int64, result *[]T) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}

	matchList := new(Matches).In("id", ids)
	return FindByMatch(c, *matchList, result, "con")
}

func save(c *Con, data Identifiable, version int, newIDs ...int64) (id int64, err *errors.Error) {
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
		if newID == 0 {
			newID = c.GenerateID()
		}
	}

	id, gErr := dbService.Save(c, data, newID, version)
	if gErr != nil {
		return 0, c.HandleWithErr(gErr)
	}
	data.SetID(id)
	return id, nil
}

// Deprecated: This method is no longer recommended. Use Save instead.
func SaveOrUpdate(c *Con, data Identifiable, newIDs ...int64) (id int64, err *errors.Error) {
	return save(c, data, 0, newIDs...)
}

func Save(c *Con, data Identifiable, newIDs ...int64) (id int64, err *errors.Error) {
	return save(c, data, 1, newIDs...)
}

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

func DeleteByMatch(c *Con, matchList []Match) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.DeleteByMatch(c, matchList)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// toSnake converts camel case to snake case
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

// GetByField queries a single record by field name and value
func GetByField[T any](c *Con, fieldName string, value interface{}, result *T) (err *errors.Error) {
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return GetByMatch(c, matchList, result)
}

// GetByCondition queries a single record by condition
func GetByCondition[T any](c *Con, condition map[string]interface{}, result *T) (err *errors.Error) {
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return GetByMatch(c, matchList, result)

}

// FindByField queries multiple records by field name and value, returns up to 1000 records
func FindByField[T any](c *Con, fieldName string, value interface{}, result *[]T, prefixes ...string) (err *errors.Error) {
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return FindByMatch(c, matchList, result, prefixes...)
}

// FindByCondition queries multiple records by condition, returns up to 1000 records
func FindByCondition[T any](c *Con, condition map[string]interface{}, result *[]T, prefixes ...string) (err *errors.Error) {
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return FindByMatch(c, matchList, result, prefixes...)
}

// FindByMatch queries multiple records by condition, returns up to 1000 records
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

// GetByMatch queries a single record by condition
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

// CountByField queries the count by field name and value
func CountByField(c *Con, fieldName string, value interface{}) (int64, *errors.Error) {
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return CountByMatch(c, matchList)
}

// CountByCondition queries the count by condition
func CountByCondition(c *Con, condition map[string]interface{}) (int64, *errors.Error) {
	matchList := make([]Match, 0, len(condition))
	for k, v := range condition {
		matchList = append(matchList, Match{Field: k, Value: v, Type: MEq})
	}
	return CountByMatch(c, matchList)
}

// CountByMatch queries the count by condition
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

// SumByMatch queries the sum by condition
func SumByMatch(c *Con, matchList []Match, field string) (float64, *errors.Error) {
	if c == nil {
		return 0, errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	sum, gErr := dbService.SumByMatch(c, matchList, field)
	if gErr != nil {
		return 0, c.HandleWithErr(gErr)
	}
	return sum, nil
}

// UpdatePart updates partial fields by ID
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

// UpdateByMatch updates records by condition
func UpdateByMatch(c *Con, matchList []Match, data map[string]interface{}) *errors.Error {
	if c == nil {
		return errors.Sys("con not init")
	}

	dbService := GetDBService(c.GetConType())

	gErr := dbService.UpdateByMatch(c, matchList, data)
	if gErr != nil {
		return c.HandleWithErr(gErr)
	}
	return nil
}

// FindByPageField queries paginated records by field name
func FindByPageField[T Identifiable](c *Con, fieldName string, value interface{}, page *load.Page, pageResp *load.PageResp, result *[]T, prefixes ...string) *errors.Error {
	matchList := []Match{{Field: fieldName, Value: value, Type: MEq}}
	return FindByPageMatch(c, matchList, page, pageResp, result, prefixes...)
}

// FindByPage queries paginated records by condition
func FindByPage[T Identifiable](c *Con, condition map[string]interface{}, page *load.Page, pageResp *load.PageResp, result *[]T, prefixes ...string) *errors.Error {
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
