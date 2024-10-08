package domainx

import (
	"github.com/jom-io/gorig/utils/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Converter[T any] struct{}

func (c *Converter[T]) FromPrimitiveD(doc interface{}) (*T, *errors.Error) {
	// 先转为primitive.D
	if doc == nil {
		return nil, errors.Sys("doc is nil")
	}
	// 将doc转为primitive.D
	docD, ok := doc.(primitive.D)
	if !ok {
		return nil, errors.Sys("doc is not primitive.D")
	}
	// 将 primitive.D 转换为 bson.M
	bsonDoc := bson.M{}
	for _, elem := range docD {
		bsonDoc[elem.Key] = elem.Value
	}
	// 解码为自定义结构体
	bytes, err := bson.Marshal(bsonDoc)
	if err != nil {
		return nil, errors.Sys("bson.Marshal error", err)
	}
	result := new(T)
	if err := bson.Unmarshal(bytes, &result); err != nil {
		return nil, errors.Sys("bson.Unmarshal error", err)
	}
	return result, nil
}

func GetEntity[T any](entity interface{}) (*T, *errors.Error) {
	converter := Converter[T]{}
	return converter.FromPrimitiveD(entity)
}

// GetListEntity 获取列表实体
func GetListEntity[T any](docList any) (*[]T, *errors.Error) {
	complexList := docList.(*[]Complex[any])
	result := make([]T, 0, len(*complexList))
	for _, doc := range *complexList {
		entity, err := GetEntity[T](doc.Data)
		if err != nil {
			return nil, err
		}
		result = append(result, *entity)
	}

	return &result, nil
}
