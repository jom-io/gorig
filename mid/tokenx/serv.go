package tokenx

import (
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/errors"
)

type GeneratorType int

const (
	Jwt = iota
)

type ManagerType int

const (
	Memory = iota
	Redis
)

const defSigning = "github.com/jom-io/gorig"

const defExpire = 3600 * 24 * 3 // 默认过期时间 3天

type TokenGenerator interface {
	Generate(userId string, userInfo map[string]interface{}, expireAt int64) (tokens string, err *errors.Error)
	Parse(token string) (*CustomClaims, *errors.Error)
}

type TokenManager interface {
	Record(userToken string, userInfo map[string]interface{}) bool
	GenerateAndRecord(userId string, userInfo map[string]interface{}, expireAt int64) (tokens string, err *errors.Error)
	IsNotExpired(token string, allowSec int64) (*CustomClaims, int)
	IsMeetRefresh(token string) bool
	Refresh(oldToken, newToken string) bool
	IsEffective(token string) bool
	Destroy(token string)
	GetUserID(token string) (userID string, exisit bool)
	CleanAll()
	Clean(userId string)
}

type TokenService struct {
	Generator TokenGenerator
	Manager   TokenManager
}

func GetDef() *TokenService {
	return Get(Jwt, Memory)
}

func Get(generatorType GeneratorType, managerType ManagerType) *TokenService {
	generator := getGenerator(generatorType)
	return &TokenService{
		Generator: getGenerator(generatorType),
		Manager:   getManager(managerType, generator),
	}
}

func getGenerator(generatorType GeneratorType) TokenGenerator {
	switch generatorType {
	case Jwt:
		return &jwtGenerator{
			SigningKey: []byte(configure.GetString("Jwt.SigningKey", defSigning)),
		}
	}
	return nil
}

func getManager(managerType ManagerType, generator TokenGenerator) TokenManager {
	switch managerType {
	case Memory:
		return &memoryImpl{
			generator: generator,
		}
	case Redis:
	// todo
	default:
		return &memoryImpl{
			generator: generator,
		}
	}
	return nil
}
