package domainx

import (
	"github.com/jom-io/gorig/utils/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Converter[T any] struct{}

func (c *Converter[T]) FromPrimitiveD(doc interface{}) (*T, *errors.Error) {
	if doc == nil {
		return nil, errors.Sys("doc is nil")
	}
	docD, ok := doc.(primitive.D)
	if !ok {
		return nil, errors.Sys("doc is not primitive.D")
	}
	bsonDoc := bson.M{}
	for _, elem := range docD {
		bsonDoc[elem.Key] = elem.Value
	}
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
