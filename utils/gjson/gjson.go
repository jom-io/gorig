package gjson

import (
	"context"
	"encoding/json"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
)

func ToBytes(obj interface{}) ([]byte, error) {
	if obj == nil {
		return []byte{}, nil
	}
	return json.Marshal(obj)
}

func MustToBytes(ctx context.Context, obj interface{}) []byte {
	bytes, err := ToBytes(obj)
	if err != nil {
		logger.Info(ctx, "MustToBytes error", zap.Error(err), zap.Any("obj", obj))
		return []byte{}
	}
	return bytes
}

func ToString(obj interface{}) (string, error) {
	bData, err := ToBytes(obj)
	if err != nil {
		return "", err
	}
	return string(bData), nil
}

func MustToString(ctx context.Context, obj interface{}) string {
	str, err := ToString(obj)
	if err != nil {
		logger.Info(ctx, "MustToString error", zap.Error(err), zap.Any("obj", obj))
		return ""
	}
	return str
}

func FromBytes[T any](bData []byte) (*T, error) {
	var obj T
	err := json.Unmarshal(bData, &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

func MustFromBytes[T any](ctx context.Context, bData []byte) *T {
	ptrObj, err := FromBytes[T](bData)
	if err != nil {
		logger.Info(ctx, "MustFromBytes error", zap.Error(err), zap.ByteString("data", bData))
		return nil
	}
	return ptrObj
}

func FromString[T any](str string) (*T, error) {
	return FromBytes[T]([]byte(str))
}

func MustFromString[T any](str string) *T {
	obj, err := FromString[T](str)
	if err != nil {
		return nil
	}
	return obj
}

func OfBytes[T any](ptr *T, bData []byte) error {
	return json.Unmarshal(bData, ptr)
}

func MustOfBytes[T any](ctx context.Context, ptr *T, bData []byte) {
	err := OfBytes[T](ptr, bData)
	if err != nil {
		logger.Info(ctx, "hyle.json.MustOfBytes", zap.Error(err), zap.ByteString("data", bData))
	}
}

func OfString[T any](ptr *T, str string) error {
	return OfBytes[T](ptr, []byte(str))
}

func MustOfString[T any](ctx context.Context, ptr *T, str string) {
	err := OfString[T](ptr, str)
	if err != nil {
		logger.Info(ctx, "hyle.json.MustOfString", zap.Error(err), zap.String("data", str))
	}
}
