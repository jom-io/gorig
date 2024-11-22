package apix

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/apix/response"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

const (
	ErrorKey = "error_g"
)

// Forcible The force parameter will decide to return a parameter check error if the value is not found.
type Forcible bool

const (
	Force    Forcible = true
	NotForce Forcible = false
)

type ParamType string

const (
	Get      ParamType = "Get"
	PostForm ParamType = "PostForm"
	PostBody ParamType = "PostBody"
)

func PutParams(ctx *gin.Context, params map[string]interface{}) {
	if ctx.IsAborted() {
		return
	}
	ctx.Set("params", params)
}

func GetParams(ctx *gin.Context, paramTypes ...ParamType) map[string]interface{} {
	if ctx.IsAborted() {
		return nil
	}
	if ctx.Keys["params"] != nil {
		return ctx.Keys["params"].(map[string]interface{})
	}
	contentType := ctx.Request.Header.Get("Content-Type")
	logger.Info(ctx, fmt.Sprintf("url: %v", ctx.Request.URL), zap.Any("method", ctx.Request.Method), zap.Any("Content-Type", contentType))
	var req map[string]interface{}
	// 读取Get中的参数
	for k, v := range ctx.Request.URL.Query() {
		if len(v) == 1 {
			if req == nil {
				req = make(map[string]interface{})
			}
			req[k] = v[0]
		} else {
			if req == nil {
				req = make(map[string]interface{})
			}
			req[k] = v
		}
	}
	if req != nil {
		logger.Info(ctx, "GetParams by url", zap.Any("req", req))
	}
	//logger.Info(ctx, "GetParams ContentLength", zap.Any("ContentLength", ctx.Request.ContentLength))
	var paramType = PostBody

	if len(paramTypes) > 0 {
		paramType = paramTypes[0]
	}

	if ctx.Request.ContentLength == 0 || paramType == Get {
		return req
	}

	isForm := strings.Contains(contentType, "application/x-www-form-urlencoded")
	if paramType == PostForm || (ctx.Request.Method == "POST" || ctx.Request.Method == "PUT") && isForm {
		err := ctx.Request.ParseForm()
		if err != nil {
			ctx.Set(ErrorKey, fmt.Sprintf("GetParams: %v", err))
			logger.Error(ctx, "GetParams by from", zap.Any("err", err))
			response.ValidatorError(ctx, err)
			return nil
		} else {
			for k, v := range ctx.Request.PostForm {
				if len(v) == 1 {
					if req == nil {
						req = make(map[string]interface{})
					}
					req[k] = v[0]
				} else {
					if req == nil {
						req = make(map[string]interface{})
					}
					req[k] = v
				}
			}
			logger.Info(ctx, "GetParams by from", zap.Any("req", req))
			if req != nil {
				return req
			}
		}
	}

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Set(ErrorKey, fmt.Sprintf("GetParams: %v", err))
		logger.Error(ctx, "GetParams by bind", zap.Any("err", err))
		response.ValidatorError(ctx, err)
		return nil
	}
	ctx.Set("params", req)
	logger.Info(ctx, "GetParams", zap.Any("req", req))
	return req
}

func BindParams(ctx *gin.Context, req interface{}) (err *errors.Error) {
	if ctx.IsAborted() {
		return errors.Verify("BindParams: ctx is aborted")
	}
	if ctx.Keys["params"] != nil {
		params := cast.ToStringMap(ctx.Keys["params"])
		jsonStr, _ := json.Marshal(params)
		if err := json.Unmarshal(jsonStr, req); err != nil {
			ctx.Set(ErrorKey, fmt.Sprintf("BindParams: %v", err))
			logger.Error(ctx, "BindParams", zap.Any("err", err))
			response.ValidatorError(ctx, err)
			return errors.Verify(fmt.Sprintf("BindParams: %v", err))
		}
		logger.Info(ctx, "BindParams", zap.Any("req", req))
		return nil
	}
	if err := ctx.ShouldBind(req); err != nil {
		ctx.Set(ErrorKey, fmt.Sprintf("BindParams: %v", err))
		logger.Error(ctx, "BindParams", zap.Any("err", err))
		response.ValidatorError(ctx, err)
		return errors.Verify(fmt.Sprintf("BindParams: %v", err))
	}
	logger.Info(ctx, "BindParams", zap.Any("req", req))
	return nil
}

func GetParamInt(ctx *gin.Context, key string, force Forcible, defValue ...int) (value int, err *errors.Error) {
	var param string
	if get, e := getParamItem(ctx, key, force); e != nil {
		if len(defValue) > 0 {
			return defValue[0], nil
		}
		return 0, e
	} else {
		param = get
	}

	if param == "" && len(defValue) > 0 {
		return defValue[0], nil
	}

	if get, e := strconv.ParseFloat(param, 64); e == nil {
		return int(get), nil
	}

	if s, e := strconv.Atoi(param); e != nil {
		fmt.Printf("%T, %v", s, s)
	} else {
		return s, nil
	}
	return 0, nil
}

func GetParamBool(ctx *gin.Context, key string, force Forcible, defValue ...bool) (value bool, err *errors.Error) {
	var param string
	if get, e := getParamItem(ctx, key, force); e != nil {
		if len(defValue) > 0 {
			return defValue[0], nil
		}
		return false, e
	} else {
		param = get
	}

	if param == "" && len(defValue) > 0 {
		return defValue[0], nil
	}

	if get, e := strconv.ParseBool(param); e == nil {
		return get, nil
	}
	return false, nil
}

func GetParamInt64(ctx *gin.Context, key string, force Forcible, defValue ...int64) (value int64, err *errors.Error) {
	var param string
	if get, e := getParamItem(ctx, key, force); e != nil {
		if len(defValue) > 0 {
			return defValue[0], nil
		}
		return 0, e
	} else {
		param = get
	}

	if param == "" && len(defValue) > 0 {
		return defValue[0], nil
	}

	if get, e := strconv.ParseFloat(param, 64); e == nil {
		return int64(get), nil
	}

	if s, e := strconv.ParseInt(param, 10, 64); e != nil {
		fmt.Printf("%T, %v", s, s)
	} else {
		return s, nil
	}
	return 0, nil
}

func GetParamFloat64(ctx *gin.Context, key string, force Forcible, defValue ...float64) (value float64, err *errors.Error) {
	var param string
	if get, e := getParamItem(ctx, key, force); e != nil {
		if len(defValue) > 0 {
			return defValue[0], nil
		}
		return 0, e
	} else {
		param = get
	}

	if param == "" && len(defValue) > 0 {
		return defValue[0], nil
	}

	if s, e := strconv.ParseFloat(param, 64); e != nil {
		fmt.Printf("%T, %v", s, s)
	} else {
		return s, nil
	}
	return 0, nil
}

func GetParamArray(ctx *gin.Context, key string, force Forcible, defValue ...[]string) (value []string, err *errors.Error) {
	if ctx.IsAborted() && ctx.GetString(ErrorKey) != "" {
		return nil, errors.Verify(ctx.GetString(ErrorKey))
	}
	req := GetParams(ctx)
	if req == nil {
		reqErr := errors.Verify("GetParam: req is nil")
		if force {
			response.ValidatorError(ctx, reqErr)
		}
		if len(defValue) > 0 {
			return defValue[0], nil
		}
		return nil, reqErr
	}
	errText := fmt.Sprintf("param: %s", key)
	defaultStr := []string{}
	if len(defValue) > 0 {
		defaultStr = defValue[0]
	}
	if value, ok := req[key]; ok {
		switch value.(type) {
		case string:
			if value.(string) == "undefined" || value.(string) == "null" {
				return defaultStr, nil
			}
			return []string{value.(string)}, nil
		case []string:
			return value.([]string), nil
		case []interface{}:
			var arr []string
			for _, v := range value.([]interface{}) {
				arr = append(arr, fmt.Sprintf("%v", v))
			}
			return arr, nil
		default:
			return nil, errors.Verify(fmt.Sprintf("GetParam: %s type error", key))
		}
	} else if force {
		ctx.Set(ErrorKey, errText)
		response.ValidatorError(ctx, errors.Verify(errText))
		return nil, errors.Verify(errText)
	} else {
		return defaultStr, nil
	}
}

func getParamItem(ctx *gin.Context, key string, force Forcible, defValue ...string) (value string, err *errors.Error) {
	if ctx.IsAborted() && ctx.GetString(ErrorKey) != "" {
		return "", errors.Verify(ctx.GetString(ErrorKey))
	}
	req := GetParams(ctx)
	if req == nil {
		reqErr := errors.Verify("GetParam: req is nil")
		if force {
			response.ValidatorError(ctx, reqErr)
			return "", reqErr
		}
		if len(defValue) > 0 {
			return defValue[0], nil
		}
		return "", nil
	}
	errText := fmt.Sprintf("param: %s", key)
	defaultStr := ""
	if len(defValue) > 0 {
		defaultStr = defValue[0]
	}
	if value, ok := req[key]; ok {
		switch value.(type) {
		case string:
			if value.(string) == "undefined" || value.(string) == "null" {
				return defaultStr, nil
			}
			return value.(string), nil
		case int:
			return fmt.Sprintf("%d", value.(int)), nil
		case int64:
			return fmt.Sprintf("%d", value.(int64)), nil
		case float64:
			return fmt.Sprintf("%.2f", value.(float64)), nil
		case bool:
			return fmt.Sprintf("%t", value.(bool)), nil
		default:
			return "", errors.Verify(fmt.Sprintf("GetParam: %s type error", key))
		}
	} else if force {
		ctx.Set(ErrorKey, errText)
		response.ValidatorError(ctx, errors.Verify(errText))
		return "", errors.Verify(errText)
	} else {
		return defaultStr, nil
	}
}

func GetParamStr(ctx *gin.Context, key string, defValue ...string) (value string, err *errors.Error) {
	// 如果defValue没有值，返回空字符串
	if len(defValue) == 0 {
		defValue = append(defValue, "")
	}
	return getParamItem(ctx, key, NotForce, defValue...)
}

func GetParam(ctx *gin.Context, key string, defValue ...string) (value interface{}, err *errors.Error) {
	if ctx.IsAborted() && ctx.GetString(ErrorKey) != "" {
		return "", errors.Verify(ctx.GetString(ErrorKey))
	}
	req := GetParams(ctx)
	if req != nil {
		if value, ok := req[key]; ok {
			return value, nil
		}
	}
	if len(defValue) > 0 {
		return defValue[0], nil
	}
	return nil, nil
}

// GetParamForce The force parameter will decide to return a parameter check error if the value is not found.
func GetParamForce(ctx *gin.Context, key string) (value string, err *errors.Error) {
	return getParamItem(ctx, key, Force)
}

func GetPageReq(ctx *gin.Context) (pageReq *load.Page, err *errors.Error) {
	page, err := GetParamInt64(ctx, "page", NotForce, 1)
	if err != nil {
		return
	}
	size, err := GetParamInt64(ctx, "size", NotForce, 10)
	if err != nil {
		return
	}
	lastID, err := GetParamInt64(ctx, "lastID", NotForce, 0)
	if err != nil {
		return
	}
	pageReq = load.BuildPage(ctx, page, size, lastID)
	return
}
