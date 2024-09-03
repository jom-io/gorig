package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"gorig/global/consts"
	"gorig/utils/errors"
	"net/http"
	"net/http/httptest"
	"strconv"
)

func Success(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, data)
}

func Fail(ctx *gin.Context, err *errors.Error) {
	httpCode := http.StatusInternalServerError
	if err.IsApplication() {
		httpCode = http.StatusBadRequest
	}
	ctx.JSON(httpCode, err)
}

func NewCtx() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	SetTraceID(ctx)
	return ctx
}

func SetTraceID(ctx *gin.Context) {
	ctx.Set(consts.TraceIDKey, xid.New().String())
}

func GetTraceID(ctx *gin.Context) string {
	return ctx.GetString(consts.TraceIDKey)
}

func GetUserID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.GetString(consts.UserID)
}

func GetUserIDInt(ctx *gin.Context) int {
	userID := GetUserID(ctx)
	if userID == "" {
		return 0
	}
	id, _ := strconv.Atoi(userID)
	return id
}

func GetUserIDInt64(ctx *gin.Context) int64 {
	userID := GetUserID(ctx)
	if userID == "" {
		return 0
	}
	id, _ := strconv.ParseInt(userID, 10, 64)
	return id
}

func SetUserID(ctx *gin.Context, userID string) {
	ctx.Set(consts.UserID, userID)
}

func GetUserInfo(ctx *gin.Context) map[string]interface{} {
	value, exists := ctx.Get(consts.UserInfo)
	if !exists {
		return nil
	}
	userInfo, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	return userInfo
}

func SetUserInfo(ctx *gin.Context, userInfo map[string]interface{}) {
	ctx.Set(consts.UserInfo, userInfo)
}

func GetUserInfoValue(ctx *gin.Context, key string) any {
	userInfo := GetUserInfo(ctx)
	if userInfo == nil {
		return nil
	}
	value, exists := userInfo[key]
	if !exists {
		return nil
	}
	return value
}

func DumpRouters(call func(info gin.RouteInfo)) {
	routes := gEngine.Routes()
	for _, route := range routes {
		call(route)
	}
}

func GetClientIP(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	clientIP := ctx.GetHeader("X-Real-IP")
	if clientIP == "" {
		clientIP = ctx.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = ctx.ClientIP()
	}
	return clientIP
}
