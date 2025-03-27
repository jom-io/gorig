package domainx

import (
	"fmt"
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/global/errc"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/gormt"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

var GormDBServ = &gormDBService{}

type gormDBService struct {
}

func init() {
	RegisterDBService(Mysql, GormDBServ)
}

var gormDbMysqlMap = make(map[string]*gorm.DB)

func UseDbConn(dbname string) *gorm.DB {
	if dbname == "" {
		logger.Logger.Error(fmt.Sprintf(errc.ErrorsDBInitFail))
		return nil
	}
	dbname = strings.ToLower(dbname)
	if _, ok := gormDbMysqlMap[dbname]; !ok {
		logger.Logger.Error(fmt.Sprintf(errc.ErrorsNotInitGlobalPointer, dbname))
		return nil
	}
	return gormDbMysqlMap[dbname]
}

func (*gormDBService) Start() error {
	sys.Info(" * DB service startup on: ", Mysql)
	sub := configure.GetSub("Mysql")
	if len(sub) > 0 {
		for k, _ := range sub {
			if configure.GetInt("Mysql."+k+".GormInit") == 1 {
				sys.Info(" * Init mysql db: ", k)
				initMysqlDB(k)
			}
		}
	}
	return nil
}

func (*gormDBService) Migrate(con *Con, tableName string, value ConTable, indexList []Index) error {
	if con.DB == nil {
		return errors.Sys("Migrate: db is nil")
	}
	if err := con.Table(tableName).AutoMigrate(value); err != nil {
		sys.Error("AutoMigrate error", err)
		logger.Logger.Fatal("AutoMigrate error", zap.Error(err))
	}
	for _, v := range indexList {
		var count int64
		con.Raw("SELECT count(1) FROM information_schema.STATISTICS WHERE table_schema = ? AND table_name = ? AND index_name = ?", con.Migrator().CurrentDatabase(), tableName, v.IdxName).Count(&count)
		if count == 0 {
			sql := "CREATE IDXTYPE% `" + v.IdxName + "` ON `" + tableName + "` ("
			for _, field := range v.Fields {
				sql += "`" + field + "`,"
			}
			sql = sql[:len(sql)-1] + ")"
			if v.IdxType == Unique {
				sql = strings.Replace(sql, "IDXTYPE%", " UNIQUE INDEX", -1)
			} else {
				sql = strings.Replace(sql, "IDXTYPE%", " INDEX", -1)
			}
			con.Exec(sql)
		}
	}

	return nil
}

func (*gormDBService) End() error {
	// 关闭数据库连接 删除数据库连接
	for k, _ := range gormDbMysqlMap {
		delete(gormDbMysqlMap, k)
	}
	sys.Info(" * Gorm service shutdown on: ", Mysql)
	return nil
}

func initMysqlDB(dbname ...string) {
	if len(dbname) > 0 {
		// 循环初始化数据库
		for _, v := range dbname {
			if dbMysql, err := gormt.GetOneMysqlClient(v); err != nil {
				logger.Logger.Fatal(fmt.Sprintf("Mysql."+v+" init fail: %s", err.Error()))
			} else {
				gormDbMysqlMap[v] = dbMysql
			}
		}
	}
}

func (s *gormDBService) GetByID(c *Con, id int64, result interface{}) error {
	if c.DB == nil {
		return fmt.Errorf("get db is nil")
	}
	//result = make(map[string]interface{})
	if err := c.DB.Table(c.TableName()).Where("id = ?", id).First(result).Error; err != nil {
		return err
	}
	return nil
}

func (s *gormDBService) Save(c *Con, data Identifiable, newID int64, version ...int) (id int64, err error) {
	var tx *gorm.DB
	if c.GetID().IsNil() {
		if newID != 0 {
			c.ID = newID
		}
		tx = c.DB.Table(c.TableName()).Create(data)
	} else {
		if version != nil && len(version) > 0 && version[0] > 0 {
			tx = c.DB.Table(c.TableName()).Where("id", data.GetID()).Updates(data)
		} else {
			if eg := c.DB.Table(c.TableName()).Where("id = ?", data.GetID()).Count(&id); eg.Error != nil {
				return 0, eg.Error
			}
			if id == 0 {
				tx = c.DB.Table(c.TableName()).Create(data)
			} else {
				tx = c.DB.Table(c.TableName()).Where("id = ?", data.GetID()).Updates(data)
			}
		}
		//tx = c.DB.Table(c.TableName()).Save(data)
	}
	if tx.Error != nil {
		return 0, tx.Error
	}
	return data.GetID().Int64(), nil
}

func (s *gormDBService) UpdatePart(c *Con, id int64, data map[string]interface{}) error {
	if err := c.DB.Table(c.TableName()).Where("id = ?", id).Updates(data).Error; err != nil {
		return err
	}
	return nil
}

func (s *gormDBService) UpdateByMatch(c *Con, matchList []Match, data map[string]interface{}) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	if err := tx.Updates(data).Error; err != nil {
		return err
	}
	return nil
}

func (s *gormDBService) Delete(c *Con, data Identifiable) error {
	if err := c.DB.Table(c.TableName()).Where("id = ?", data.GetID()).Delete(&data).Error; err != nil {
		return err
	}
	return nil
}

func (s *gormDBService) DeleteByMatch(c *Con, matchList []Match) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	if err := tx.Delete(&Options{}).Error; err != nil {
		return err
	}
	return tx.Error
}

var mysqlKeywords = []string{
	"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "MERGE", "REPLACE",
	"DESCRIBE", "EXPLAIN", "SHOW", "GRANT", "REVOKE", "USE", "LOCK", "UNLOCK", "SET", "COMMIT", "ROLLBACK",
}

func matchMysqlCond(matchList []Match, tx *gorm.DB) {
	for _, match := range matchList {
		if v, ok := match.Value.(ValueField); ok && !v.Check(mysqlKeywords...) {
			continue
		}
		var condition string
		switch match.Type {
		case MEq:
			condition = "="
		case MLt:
			condition = "<"
		case MLte:
			condition = "<="
		case MGt:
			condition = ">"
		case MGte:
			condition = ">="
		case MNE:
			condition = "!="
		case MLIKE:
			tx = tx.Where(match.Field+" like ?", "%"+match.Value.(string)+"%")
			continue
		case MIN:
			tx = tx.Where(match.Field+" in (?)", match.Value)
			continue
		case MNOTIN:
			tx = tx.Where(match.Field+" not in (?)", match.Value)
			continue
		case MNEmpty:
			tx = tx.Where(match.Field + " != '' and " + match.Field + " is not null")
			continue
		case Near:
			near := match.ToNearMatch()
			tx = tx.Select("*, (6371 * acos(cos(radians(?)) * cos(radians("+near.LatField+")) * cos(radians("+near.LngField+") - radians(?)) + sin(radians(?)) * sin(radians("+near.LatField+")))) AS distance", near.Lat, near.Lng, near.Lat)
			if near.Distance > 0 {
				tx = tx.Where("6371 * acos(cos(radians(?)) * cos(radians("+near.LatField+")) * cos(radians("+near.LngField+") - radians(?)) + sin(radians(?)) * sin(radians("+near.LatField+"))) < ?", near.Lat, near.Lng, near.Lat, near.Distance)
			}
			tx = tx.Order("distance")
			continue
		default:
			tx = tx.Where(match.Field+" = ?", match.Value)
			continue
		}

		if fieldValue, ok := match.Value.(ValueField); ok {
			tx = tx.Where(gorm.Expr(match.Field + " " + condition + " " + string(fieldValue)))
		} else {
			tx = tx.Where(match.Field+" "+condition+" ?", match.Value)
		}
	}
}

func sortMysqlCond(sortList Sorts, tx *gorm.DB) {
	if len(sortList) > 0 {
		for _, v := range sortList {
			desc := ""
			if !v.Asc {
				desc = " desc"
			}
			tx = tx.Order(v.Field + desc)
		}
	}
}

func (s *gormDBService) FindByMatch(c *Con, matchList []Match, result interface{}, prefixes ...string) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	sortMysqlCond(c.Sort, tx)
	if err := tx.Limit(10000).Find(result).Error; err != nil {
		return err
	}
	return tx.Error
}

func (s *gormDBService) GetByMatch(c *Con, matchList []Match, result interface{}) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	sortMysqlCond(c.Sort, tx)
	if err := tx.First(result).Error; err != nil {
		return err
	}
	return tx.Error
}

func (s *gormDBService) CountByMatch(c *Con, matchList []Match) (int64, error) {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *gormDBService) SumByMatch(c *Con, matchList []Match, field string) (float64, error) {
	tx := c.DB.Table(c.TableName())
	if !Check(field) {
		return 0, errors.Sys(fmt.Sprintf("field is not valid: %s", field))
	}
	matchMysqlCond(matchList, tx)
	var sum *float64
	if err := tx.Select("sum(" + field + ")").Scan(&sum).Error; err != nil {
		return 0, err
	}
	if sum == nil {
		return 0, nil
	}
	return *sum, nil
}

func (s *gormDBService) FindByPageMatch(c *Con, matchList []Match, page *load.Page, total *load.Total, result interface{}, prefixes ...string) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	sortMysqlCond(c.Sort, tx)
	count := int64(0)
	if err := tx.Model(result).Count(&count).Error; err != nil {
		return err
	}
	if page.LastID > 0 {
		tx = tx.Where("id < ?", page.LastID).Order("id desc").Limit(int(page.Size)).Find(result)
	} else {
		tx = tx.Order("id desc").Limit(int(page.Size)).Offset(int(page.Offset())).Find(result)
	}
	total.Set(count)
	return tx.Error
}
