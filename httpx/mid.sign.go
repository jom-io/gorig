package httpx

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix/response"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/mid/tokenx"
	"github.com/spf13/cast"
	"strings"
)

type HeaderParams struct {
	Authorization string `header:"Authorization" binding:"required,min=20"`
}

func SignDef() gin.HandlerFunc {
	return sign(tokenx.Memory, nil)
}

func SignRedis() gin.HandlerFunc {
	return sign(tokenx.Redis, nil)
}

func SignUserDef(userFilter map[string]interface{}) gin.HandlerFunc {
	return sign(tokenx.Memory, userFilter)
}

func SignUserRedis(userFilter map[string]interface{}) gin.HandlerFunc {
	return sign(tokenx.Redis, userFilter)
}

func sign(merType tokenx.ManagerType, userFilter map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := GetTokenByCtx(c, true)
		if token == "" {
			return
		}
		get := tokenx.Get(tokenx.Jwt, merType)
		userID, exisit := get.Manager.GetUserID(token)
		if exisit {
			c.Set(consts.TokenKey, token)
			SetUserID(c, userID)
			if claims, err := get.Generator.Parse(token); err != nil {
				response.ErrorTokenAuthFail(c)
			} else if !filterUserInfo(claims.UserInfo, userFilter) {
				response.ErrorForbidden(c)
			} else {
				SetUserInfo(c, claims.UserInfo)
				c.Next()
			}
		} else {
			response.ErrorTokenAuthFail(c)
		}
	}
}

func filterUserInfo(userInfo map[string]interface{}, filter map[string]interface{}) bool {
	//logger.Info(nil, "filterUserInfo", zap.Any("userInfo", userInfo), zap.Any("filter", filter))
	if userInfo == nil || filter == nil {
		return true
	}
	for k, v := range filter {
		if v == nil {
			continue
		}
		switch v {
		case consts.NotNull:
			if _, exists := userInfo[k]; !exists {
				return false
			}
			if userInfo[k] == nil || userInfo[k] == "" {
				return false
			}
		default:
			userValue := cast.ToString(userInfo[k])
			if strings.Contains(userValue, ",") {
				if !strings.Contains(userValue, cast.ToString(v)) {
					return false
				}
			} else if userValue != v {
				return false
			}
		}
	}
	return true
}

//func notNull(m map[string]interface{}, key string) bool {
//	if v, exists := m[key]; exists {
//		return v != nil && v != ""
//	}
//	return false
//}

func GetToken(c *gin.Context) string {
	return c.GetString(consts.TokenKey)
}

func GetUserIDByToken(token string) (userID string) {
	if id, exisit := tokenx.GetDef().Manager.GetUserID(token); exisit {
		return id
	}
	return ""
}

// GetTokenByCtx
func GetTokenByCtx(c *gin.Context, mustExit bool) string {
	headerParams := HeaderParams{}
	if err := c.ShouldBindHeader(&headerParams); err != nil {
		if mustExit {
			response.TokenErrorParam(c, fmt.Sprintf("%s header: %s", consts.JwtTokenMustValid, err.Error()))
		}
		return ""
	}
	split := strings.Split(headerParams.Authorization, " ")
	if len(split) != 2 || split[0] != "Bearer" || len(split[1]) < 20 {
		if mustExit {
			response.ErrorTokenBaseInfo(c)
		}
		return ""
	}
	token := split[1]
	return token
}
