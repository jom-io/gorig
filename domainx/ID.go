package domainx

import (
	"math/rand"
	"time"
)

type ID int64

func (i *ID) GenerateID() int64 {
	if i == nil {
		i = new(ID)
	}
	// 创建一个新的随机数生成器，使用当前时间戳作为种子
	randSource := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(randSource)
	// 获取当前时间戳的毫秒表示形式
	currentTimestampMillis := time.Now().UnixNano() / 1_000_000
	// 生成3位随机数
	randomNumber := rnd.Intn(900) + 100
	// 将当前时间戳和3位随机数组合起来
	result := currentTimestampMillis*1000 + int64(randomNumber)
	return result
}

func (i *ID) New() {
	i = new(ID)
}

func (i *ID) Reset() {
	if i == nil {
		i = new(ID)
	}
	i.SetID(0)
}

func (i *ID) Generate() int64 {
	i.SetID(i.GenerateID())
	return i.Int64()
}

func (i *ID) SetID(id int64) {
	if i == nil {
		i = new(ID)
	} else {
		*i = ID(id)
	}
}

func (i *ID) Int64() int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
}

func (i *ID) IsNil() bool {
	return i == nil || i.IsZero()
}

func (i *ID) IsZero() bool {
	return i != nil && i.Int64() == 0
}

func (i *ID) NotNil() bool {
	return !i.IsZero()
}

func (i *ID) NotZero() bool {
	return i != nil && i.Int64() != 0
}

func (i *ID) Equal(id int64) bool {
	return i.Int64() == id
}
