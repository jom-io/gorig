package domainx

import (
	"fmt"
	"go.uber.org/zap"
	"gorig/apix/load"
	"gorig/global/errc"
	configure "gorig/utils/cofigure"
	"gorig/utils/errors"
	"gorig/utils/gormt"
	"gorig/utils/logger"
	"gorig/utils/sys"
	"gorm.io/gorm"
	"strings"
)

var gormDBServ = &GormDBService{}

type GormDBService struct {
}

func init() {
	RegisterDBService(Mysql, gormDBServ)
}

var gormDbMysqlMap = make(map[string]*gorm.DB)

func UseDbConn(dbname string) *gorm.DB {
	if dbname == "" {
		logger.Logger.Error(fmt.Sprintf(errc.ErrorsDBInitFail))
		return nil
	}
	//GormDbMysqlMap 如果在配置文件中配置了多个数据库连接，可以通过这个map来获取
	// 校验是否存在该数据库连接
	dbname = strings.ToLower(dbname)
	if _, ok := gormDbMysqlMap[dbname]; !ok {
		logger.Logger.Error(fmt.Sprintf(errc.ErrorsNotInitGlobalPointer, dbname))
		return nil
	}
	return gormDbMysqlMap[dbname]
}

func printSql(tag string, gormDB *gorm.DB) {
	if !sys.RunMode.IsProd() {
		logger.Info(nil, tag, zap.String("sql", gormDB.Dialector.Explain(gormDB.Statement.SQL.String(), gormDB.Statement.Vars...)))
	}
}

func (c *Con) BeforeCreate(gormDB *gorm.DB) error {
	//logger.Logger.Info("BeforeCreate")
	c.DB = gormDB
	// 此处可用于统一控制执行超时时间
	//后续的代码就可以像普通业务 model 一样操作，
	// b.Exec(sql,参数1，参数2，...)
	//b.Raw(sql,参数1，参数2，...)
	return nil
}

func (c *Con) AfterCreate(gormDB *gorm.DB) error {
	printSql("AfterCreate", gormDB)
	return nil
}

// BeforeUpdate BeforeUpdate、BeforeSave 函数都会因为 更新类的操作而被触发
// 如果baseModel 和 普通业务 model 都想使用回调函数，那么请设置不同的回调函数名，例如：这里设置 BeforeUpdate、普通业务model 设置 BeforeSave 即可
func (c *Con) BeforeUpdate(gormDB *gorm.DB) error {
	//第一步必须反向将 gormDB 赋值给 b.DB
	//logger.Logger.Info("BeforeUpdate")
	c.DB = gormDB
	//后续的代码就可以像普通业务 model 一样操作，
	// b.Exec(sql,参数1，参数2，...)
	//b.Raw(sql,参数1，参数2，...)
	return nil
}

func (c *Con) AfterUpdate(gormDB *gorm.DB) error {
	printSql("AfterUpdate", gormDB)
	return nil
}

func (*Con) HandleError(tx *gorm.DB) (err *errors.Error) {
	if tx.Error != nil {
		err = errors.Sys("数据库操作失败", tx.Error)
		return err
	}
	return nil
}

func (*GormDBService) start() error {
	sys.Info(" * DB service startup on: ", Mysql)
	sub := configure.GetSub("Mysql")
	if len(sub) > 0 {
		for k, _ := range sub {
			// 读取配置中的IsInitGlobalGormMysql字段
			if configure.GetInt("Mysql."+k+".GormInit") == 1 {
				sys.Info(" * Init mysql db: ", k)
				initMysqlDB(k)
			}
		}
	}
	return nil
}

func (s *GormDBService) Start() error {
	return s.start()
}

func (*GormDBService) migrate(con *Con, tableName string, value ConTable, indexList []Index) error {
	if con.DB == nil {
		return errors.Sys("Migrate: db is nil")
	}
	if err := con.Table(tableName).AutoMigrate(value); err != nil {
		sys.Error("AutoMigrate error", err)
		logger.Logger.Fatal("AutoMigrate error", zap.Error(err))
	}
	// 如果索引不存在则创建索引
	for _, v := range indexList {
		// 查询索引是否存在
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

	// 删除create_at和update_at字段
	//con.Exec("ALTER TABLE `" + tableName + "` DROP COLUMN `create_at`")
	//con.Exec("ALTER TABLE `" + tableName + "` DROP COLUMN `update_at`")
	// 如果字段类型不是datetime(0)则修改字段类型
	// 查询目前该表的字段类型
	//var columnType string
	//db.Raw("SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE table_schema = ? AND table_name = ? AND column_name = ?", db.Migrator().CurrentDatabase(), tableName, "created_at").Scan(&columnType)
	//if columnType != "datetime(0)" {
	//	logger.Info(nil, "AutoMigrate Change created_at, columnType: "+columnType)
	//	db.Exec("ALTER TABLE `" + tableName + "` CHANGE `created_at` `created_at` DATETIME(0)  NULL  DEFAULT NULL;")
	//}
	//db.Raw("SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE table_schema = ? AND table_name = ? AND column_name = ?", db.Migrator().CurrentDatabase(), tableName, "updated_at").Scan(&columnType)
	//if columnType != "datetime(0)" {
	//	logger.Info(nil, "AutoMigrate Change updated_at")
	//	db.Exec("ALTER TABLE `" + tableName + "` CHANGE `updated_at` `updated_at` DATETIME(0)  NULL  DEFAULT NULL;")
	//}

	return nil
}

func (*GormDBService) end() error {
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

func (s *GormDBService) GetByID(c *Con, id int64, result interface{}) error {
	if c.DB == nil {
		return fmt.Errorf("get db is nil")
	}
	if err := c.DB.Table(c.TableName()).First(result, id).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormDBService) Save(c *Con, data load.Identifiable, newID int64) (id int64, err error) {
	if c.ID == 0 && newID > 0 {
		c.ID = newID
	}

	tx := c.DB.Table(c.TableName()).Save(data)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return data.GetID(), nil
}

func (s *GormDBService) UpdatePart(c *Con, id int64, data map[string]interface{}) error {
	if err := c.DB.Table(c.TableName()).Where("id = ?", id).Updates(data).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormDBService) Delete(c *Con, id int64) error {
	if err := c.DB.Table(c.TableName()).Delete(id).Error; err != nil {
		return err
	}
	return nil
}

// matchMysqlCond Mysql根据条件列表获取条件
func matchMysqlCond(matchList []Match, tx *gorm.DB) {
	for _, match := range matchList {
		switch match.Type {
		case MEq:
			tx = tx.Where(match.Field+" = ?", match.Value)
		case MLt:
			tx = tx.Where(match.Field+" < ?", match.Value)
		case MLte:
			tx = tx.Where(match.Field+" <= ?", match.Value)
		case MGt:
			tx = tx.Where(match.Field+" > ?", match.Value)
		case MGte:
			tx = tx.Where(match.Field+" >= ?", match.Value)
		case MLIKE:
			tx = tx.Where(match.Field+" like ?", "%"+match.Value.(string)+"%")
		case MNE:
			tx = tx.Where(match.Field+" != ?", match.Value)
		case MIN:
			tx = tx.Where(match.Field+" in (?)", match.Value)
		case MNOTIN:
			tx = tx.Where(match.Field+" not in (?)", match.Value)
		default:
			tx = tx.Where(match.Field+" = ?", match.Value)
		}
	}
}

func (s *GormDBService) FindByMatch(c *Con, matchList []Match, result interface{}, prefixes ...string) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	if err := tx.Limit(1000).Find(result).Error; err != nil {
		return err
	}
	return tx.Error
}

func (s *GormDBService) GetByMatch(c *Con, matchList []Match, result interface{}) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	if err := tx.First(result).Error; err != nil {
		return err
	}
	return tx.Error
}

func (s *GormDBService) CountByMatch(c *Con, matchList []Match) (int64, error) {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *GormDBService) FindByPageMatch(c *Con, matchList []Match, page *load.Page, pageResp *load.PageResp, result interface{}, prefixes ...string) error {
	tx := c.DB.Table(c.TableName())
	matchMysqlCond(matchList, tx)
	if err := tx.Count(&pageResp.Total).Error; err != nil {
		return err
	}
	if page.LastID > 0 {
		tx = tx.Where("id < ?", page.LastID).Order("id desc").Limit(int(page.Size)).Find(result)
	} else {
		tx = tx.Order("id desc").Limit(int(page.Size)).Offset(int(page.Offset())).Find(result)
	}

	return tx.Error
}
