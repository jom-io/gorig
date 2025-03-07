package gormt

import (
	"github.com/jom-io/gorig/global/errc"
	"github.com/jom-io/gorig/utils/logger"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"time"
)

func MaskNotDataError(gormDB *gorm.DB) {
	gormDB.Statement.RaiseErrorOnNotFound = false
}

func CreateBeforeHook(gormDB *gorm.DB) {
	if reflect.TypeOf(gormDB.Statement.Dest).Kind() != reflect.Ptr {
		logger.Logger.Error(errc.ErrorsGormDBCreateParamsNotPtr)
	} else {
		destValueOf := reflect.ValueOf(gormDB.Statement.Dest).Elem()
		if destValueOf.Type().Kind() == reflect.Slice || destValueOf.Type().Kind() == reflect.Array {
			inLen := destValueOf.Len()
			for i := 0; i < inLen; i++ {
				row := destValueOf.Index(i)
				if row.Type().Kind() == reflect.Struct {
					if b, column := structHasSpecialField("CreatedAt", row); b {
						destValueOf.Index(i).FieldByName(column).Set(reflect.ValueOf(time.Now().Format(time.DateTime)))
					}
					if b, column := structHasSpecialField("UpdatedAt", row); b {
						destValueOf.Index(i).FieldByName(column).Set(reflect.ValueOf(time.Now().Format(time.DateTime)))
					}

				} else if row.Type().Kind() == reflect.Map {
					if b, column := structHasSpecialField("created_at", row); b {
						row.SetMapIndex(reflect.ValueOf(column), reflect.ValueOf(time.Now().Format(time.DateTime)))
					}
					if b, column := structHasSpecialField("updated_at", row); b {
						row.SetMapIndex(reflect.ValueOf(column), reflect.ValueOf(time.Now().Format(time.DateTime)))
					}
				}
			}
		} else if destValueOf.Type().Kind() == reflect.Struct {
			if b, column := structHasSpecialField("CreatedAt", gormDB.Statement.Dest); b {
				gormDB.Statement.SetColumn(column, time.Now().Format(time.DateTime))
			}
			if b, column := structHasSpecialField("UpdatedAt", gormDB.Statement.Dest); b {
				gormDB.Statement.SetColumn(column, time.Now().Format(time.DateTime))
			}
		} else if destValueOf.Type().Kind() == reflect.Map {
			if b, column := structHasSpecialField("created_at", gormDB.Statement.Dest); b {
				destValueOf.SetMapIndex(reflect.ValueOf(column), reflect.ValueOf(time.Now().Format(time.DateTime)))
			}
			if b, column := structHasSpecialField("updated_at", gormDB.Statement.Dest); b {
				destValueOf.SetMapIndex(reflect.ValueOf(column), reflect.ValueOf(time.Now().Format(time.DateTime)))
			}
		}
	}
}

func UpdateBeforeHook(gormDB *gorm.DB) {
	if reflect.TypeOf(gormDB.Statement.Dest).Kind() == reflect.Struct {
		//_ = gormDB.AddError(errors.New(my_errors.ErrorsGormDBUpdateParamsNotPtr))
		logger.Logger.Error(errc.ErrorsGormDBUpdateParamsNotPtr)
	} else if reflect.TypeOf(gormDB.Statement.Dest).Kind() == reflect.Map {
		// 如果是调用了 gorm.Update 、updates 函数 , 在参数没有传递指针的情况下，无法触发回调函数

	} else if reflect.TypeOf(gormDB.Statement.Dest).Kind() == reflect.Ptr && reflect.ValueOf(gormDB.Statement.Dest).Elem().Kind() == reflect.Struct {
		if b, column := structHasSpecialField("UpdatedAt", gormDB.Statement.Dest); b {
			gormDB.Statement.SetColumn(column, time.Now().Format(time.DateTime))
		}
	} else if reflect.TypeOf(gormDB.Statement.Dest).Kind() == reflect.Ptr && reflect.ValueOf(gormDB.Statement.Dest).Elem().Kind() == reflect.Map {
		if b, column := structHasSpecialField("updated_at", gormDB.Statement.Dest); b {
			destValueOf := reflect.ValueOf(gormDB.Statement.Dest).Elem()
			destValueOf.SetMapIndex(reflect.ValueOf(column), reflect.ValueOf(time.Now().Format(time.DateTime)))
		}
	}
}

func structHasSpecialField(fieldName string, anyStructPtr interface{}) (bool, string) {
	var tmp reflect.Type
	if reflect.TypeOf(anyStructPtr).Kind() == reflect.Ptr && reflect.ValueOf(anyStructPtr).Elem().Kind() == reflect.Map {
		destValueOf := reflect.ValueOf(anyStructPtr).Elem()
		for _, item := range destValueOf.MapKeys() {
			if item.String() == fieldName {
				return true, fieldName
			}
		}
	} else if reflect.TypeOf(anyStructPtr).Kind() == reflect.Ptr && reflect.ValueOf(anyStructPtr).Elem().Kind() == reflect.Struct {
		destValueOf := reflect.ValueOf(anyStructPtr).Elem()
		tf := destValueOf.Type()
		for i := 0; i < tf.NumField(); i++ {
			if !tf.Field(i).Anonymous && tf.Field(i).Type.Kind() != reflect.Struct {
				if tf.Field(i).Name == fieldName {
					return true, getColumnNameFromGormTag(fieldName, tf.Field(i).Tag.Get("gorm"))
				}
			} else if tf.Field(i).Type.Kind() == reflect.Struct {
				tmp = tf.Field(i).Type
				for j := 0; j < tmp.NumField(); j++ {
					if tmp.Field(j).Name == fieldName {
						return true, getColumnNameFromGormTag(fieldName, tmp.Field(j).Tag.Get("gorm"))
					}
				}
			}
		}
	} else if reflect.Indirect(anyStructPtr.(reflect.Value)).Type().Kind() == reflect.Struct {
		destValueOf := anyStructPtr.(reflect.Value)
		tf := destValueOf.Type()
		for i := 0; i < tf.NumField(); i++ {
			if !tf.Field(i).Anonymous && tf.Field(i).Type.Kind() != reflect.Struct {
				if tf.Field(i).Name == fieldName {
					return true, getColumnNameFromGormTag(fieldName, tf.Field(i).Tag.Get("gorm"))
				}
			} else if tf.Field(i).Type.Kind() == reflect.Struct {
				tmp = tf.Field(i).Type
				for j := 0; j < tmp.NumField(); j++ {
					if tmp.Field(j).Name == fieldName {
						return true, getColumnNameFromGormTag(fieldName, tmp.Field(j).Tag.Get("gorm"))
					}
				}
			}
		}
	} else if reflect.Indirect(anyStructPtr.(reflect.Value)).Type().Kind() == reflect.Map {
		destValueOf := anyStructPtr.(reflect.Value)
		for _, item := range destValueOf.MapKeys() {
			if item.String() == fieldName {
				return true, fieldName
			}
		}
	}
	return false, ""
}

func getColumnNameFromGormTag(defaultColumn, TagValue string) (str string) {
	pos1 := strings.Index(TagValue, "column:")
	if pos1 == -1 {
		str = defaultColumn
		return
	} else {
		TagValue = TagValue[pos1+7:]
	}
	pos2 := strings.Index(TagValue, ";")
	if pos2 == -1 {
		str = TagValue
	} else {
		str = TagValue[:pos2]
	}
	return strings.ReplaceAll(str, " ", "")
}
