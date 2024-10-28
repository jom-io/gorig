package domainx

import (
	"fmt"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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
		err = errors.Sys(fmt.Sprintf("数据库操作失败: %v", tx.Error))
		return err
	}
	return nil
}
