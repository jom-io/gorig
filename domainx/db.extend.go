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
	c.DB = gormDB
	return nil
}

func (c *Con) AfterCreate(gormDB *gorm.DB) error {
	printSql("AfterCreate", gormDB)
	return nil
}

func (c *Con) BeforeUpdate(gormDB *gorm.DB) error {
	c.DB = gormDB
	return nil
}

func (c *Con) AfterUpdate(gormDB *gorm.DB) error {
	printSql("AfterUpdate", gormDB)
	return nil
}

func (*Con) HandleError(tx *gorm.DB) (err *errors.Error) {
	if tx.Error != nil {
		err = errors.Sys(fmt.Sprintf("sql error: %v", tx.Error))
		return err
	}
	return nil
}
