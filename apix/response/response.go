package response

import (
	"encoding/json"
	"fmt"
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
	// 将结构体序列化为 JSON 字符串
	marshal, err := json.Marshal(g)
	if err != nil {
		logger.Error(nil, err.Error())
		return g
	}
	// 判断是否包含下划线
	if !strings.Contains(string(marshal), "_") {
		return g
	}

	logger.Info(nil, string(marshal))
	// 将 JSON 字符串反序列化为 map[string]interface{}
	var data map[string]interface{}
	if err := json.Unmarshal(marshal, &data); err != nil {
		return g
	}

	// 创建一个正则表达式，用于匹配下划线后的字符
	re := regexp.MustCompile(`_([a-z])`)

	// 创建一个新的 map 用于存储驼峰风格的键
	camelData := make(map[string]interface{})
	for k, v := range data {
		// 将下划线后的字母转为大写
		camelKey := re.ReplaceAllStringFunc(k, func(s string) string {
			return strings.ToUpper(strings.TrimPrefix(s, "_"))
		})

		// 添加到新的 map
		camelData[camelKey] = v
	}
	logger.Info(nil, fmt.Sprintf("转换前的数据：%v", data))
	logger.Info(nil, fmt.Sprintf("转换后的数据：%v", camelData))

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
	// 判断是否调用过JSON，如果调用过则不再调用
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

// ReturnJsonFromString 将json字符窜以标准json格式返回（例如，从redis读取json格式的字符串，返回给浏览器json格式）
func ReturnJsonFromString(context *gin.Context, httpCode int, jsonStr string) {
	context.Header("Content-Type", "application/json; charset=utf-8")
	context.String(httpCode, jsonStr)
}

// 语法糖函数封装
func S(c *gin.Context) {
	ReturnJson(c, http.StatusOK, consts.CurdStatusOkCode, consts.CurdStatusOkMsg, nil)
}

// Success 直接返回成功
func Success(c *gin.Context, msg string, data *interface{}) {
	if msg == "" {
		msg = consts.CurdStatusOkMsg
	}
	ReturnJson(c, http.StatusOK, consts.CurdStatusOkCode, msg, data)
	c.Abort()
}

// Fail 失败的业务逻辑
func Fail(c *gin.Context, dataCode int, msg string, data *interface{}) {
	ReturnJson(c, http.StatusBadRequest, dataCode, msg, data)
	c.Abort()
}

// ErrorTokenBaseInfo token 基本的格式错误
func ErrorTokenBaseInfo(c *gin.Context) {
	ReturnJson(c, http.StatusBadRequest, http.StatusBadRequest, errc.ErrorsTokenBaseInfo, nil)
	c.Abort()
}

// ErrorTokenAuthFail token 权限校验失败
func ErrorTokenAuthFail(c *gin.Context) {
	ReturnJson(c, http.StatusUnauthorized, http.StatusUnauthorized, errc.ErrorsNoAuthorization, nil)
	c.Abort()
}

// ErrorForbidden token 权限身份校验失败
func ErrorForbidden(c *gin.Context) {
	ReturnJson(c, http.StatusForbidden, http.StatusForbidden, errc.ErrorsTokenPermissionDenied, nil)
	c.Abort()
}

func ErrorServiceForbidden(c *gin.Context) {
	ReturnJson(c, http.StatusForbidden, http.StatusForbidden, errc.ErrorsServicePermissionDenied, nil)
	c.Abort()
}

// ErrorTokenRefreshFail token不符合刷新条件
func ErrorTokenRefreshFail(c *gin.Context) {
	ReturnJson(c, http.StatusUnauthorized, http.StatusUnauthorized, errc.ErrorsRefreshTokenFail, nil)
	c.Abort()
}

// TokenErrorParam 参数校验错误
func TokenErrorParam(c *gin.Context, msg string) {
	ReturnJson(c, http.StatusForbidden, consts.ValidatorParamsCheckFailCode, msg, nil)
	c.Abort()
}

// ErrorCasbinAuthFail 鉴权失败，返回 405 方法不允许访问
func ErrorCasbinAuthFail(c *gin.Context, msg interface{}) {
	ReturnJson(c, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, errc.ErrorsCasbinNoAuthorization, &msg)
	c.Abort()
}

// ErrorParam 参数校验错误
func ErrorParam(c *gin.Context, wrongParam interface{}) {
	ReturnJson(c, http.StatusBadRequest, consts.ValidatorParamsCheckFailCode, consts.ValidatorParamsCheckFailMsg, &wrongParam)
	c.Abort()
}

// ErrorSystem 系统执行代码错误
func ErrorSystem(c *gin.Context, msg string, data interface{}) {
	ReturnJson(c, http.StatusInternalServerError, consts.ServerOccurredErrorCode, consts.ServerOccurredErrorMsg+msg, &data)
	c.Abort()
}

// StatusTooManyRequests 请求过于频繁
func ErrorTooManyRequests(c *gin.Context) {
	ReturnJson(c, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), nil)
	c.Abort()
}

// ValidatorError 翻译表单参数验证器出现的校验错误
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
