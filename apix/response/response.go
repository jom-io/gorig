package response

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/global/errc"
	"github.com/jom-io/gorig/utils/logger"
	"net/http"
	"regexp"
	"strings"
)

const Tocamel = "toCamel"

func toCamel(g *interface{}) any {
	if g == nil {
		return g
	}
	marshal, err := json.Marshal(g)
	if err != nil {
		logger.Error(nil, err.Error())
		return g
	}
	if !strings.Contains(string(marshal), "_") {
		return g
	}

	logger.Info(nil, string(marshal))
	var data map[string]interface{}
	if err := json.Unmarshal(marshal, &data); err != nil {
		return g
	}

	re := regexp.MustCompile(`_([a-z])`)

	camelData := make(map[string]interface{})
	for k, v := range data {
		camelKey := re.ReplaceAllStringFunc(k, func(s string) string {
			return strings.ToUpper(strings.TrimPrefix(s, "_"))
		})
		camelData[camelKey] = v
	}
	return camelData
}

func SetToCamel(c *gin.Context) {
	c.Set(Tocamel, true)
}

func GetToCamel(c *gin.Context) bool {
	if v, exists := c.Get(Tocamel); exists {
		return v.(bool)
	}
	return false
}

func ReturnJson(context *gin.Context, httpCode int, dataCode int, msg string, data *interface{}) {
	if context.Writer.Written() {
		return
	}
	result := gin.H{
		"code": dataCode,
		"msg":  msg,
	}
	if exists := GetToCamel(context); exists {
		result["data"] = toCamel(data)
	} else {
		result["data"] = data
	}
	context.JSON(httpCode, result)
}

func ReturnJsonFromString(context *gin.Context, httpCode int, jsonStr string) {
	context.Header("Content-Type", "application/json; charset=utf-8")
	context.String(httpCode, jsonStr)
}

func S(c *gin.Context) {
	ReturnJson(c, http.StatusOK, consts.CurdStatusOkCode, consts.CurdStatusOkMsg, nil)
}

func Success(c *gin.Context, msg string, data *interface{}) {
	if msg == "" {
		msg = consts.CurdStatusOkMsg
	}
	ReturnJson(c, http.StatusOK, consts.CurdStatusOkCode, msg, data)
	c.Abort()
}

func Fail(c *gin.Context, dataCode int, msg string, data *interface{}) {
	ReturnJson(c, http.StatusBadRequest, dataCode, msg, data)
	c.Abort()
}

// ErrorTokenBaseInfo token 400
func ErrorTokenBaseInfo(c *gin.Context) {
	ReturnJson(c, http.StatusBadRequest, http.StatusBadRequest, errc.ErrorsTokenBaseInfo, nil)
	c.Abort()
}

// ErrorTokenAuthFail token 401
func ErrorTokenAuthFail(c *gin.Context) {
	ReturnJson(c, http.StatusUnauthorized, http.StatusUnauthorized, errc.ErrorsNoAuthorization, nil)
	c.Abort()
}

// ErrorForbidden token 403
func ErrorForbidden(c *gin.Context) {
	ReturnJson(c, http.StatusForbidden, http.StatusForbidden, errc.ErrorsTokenPermissionDenied, nil)
	c.Abort()
}

// ErrorServiceForbidden  403
func ErrorServiceForbidden(c *gin.Context) {
	ReturnJson(c, http.StatusForbidden, http.StatusForbidden, errc.ErrorsServicePermissionDenied, nil)
	c.Abort()
}

// ErrorTokenRefreshFail 401
func ErrorTokenRefreshFail(c *gin.Context) {
	ReturnJson(c, http.StatusUnauthorized, http.StatusUnauthorized, errc.ErrorsRefreshTokenFail, nil)
	c.Abort()
}

// TokenErrorParam params 403
func TokenErrorParam(c *gin.Context, msg string) {
	ReturnJson(c, http.StatusForbidden, consts.ValidatorParamsCheckFailCode, msg, nil)
	c.Abort()
}

// ErrorCasbinAuthFail 405
func ErrorCasbinAuthFail(c *gin.Context, msg interface{}) {
	ReturnJson(c, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, errc.ErrorsCasbinNoAuthorization, &msg)
	c.Abort()
}

// ErrorParam params 403
func ErrorParam(c *gin.Context, wrongParam interface{}) {
	ReturnJson(c, http.StatusBadRequest, consts.ValidatorParamsCheckFailCode, consts.ValidatorParamsCheckFailMsg, &wrongParam)
	c.Abort()
}

// ErrorSystem 500
func ErrorSystem(c *gin.Context, msg string, data interface{}) {
	ReturnJson(c, http.StatusInternalServerError, consts.ServerOccurredErrorCode, consts.ServerOccurredErrorMsg+" "+msg, &data)
	c.Abort()
}

// StatusTooManyRequests 429
func ErrorTooManyRequests(c *gin.Context) {
	ReturnJson(c, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), nil)
	c.Abort()
}

// ValidatorError 400
func ValidatorError(c *gin.Context, err error) {
	//if errs, ok := err.(validator.ValidationErrors); ok {
	//	var myInterface interface{} = errs.Translate
	//	ReturnJson(c, http.StatusBadRequest, consts.ValidatorParamsCheckFailCode, consts.ValidatorParamsCheckFailMsg, &myInterface)
	//} else {
	var tips interface{} = err.Error()
	logger.Error(c, err.Error())
	ReturnJson(c, http.StatusBadRequest, consts.ValidatorParamsCheckFailCode, consts.ValidatorParamsCheckFailMsg, &tips)
	//}
	c.Abort()
}
