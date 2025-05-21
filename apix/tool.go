package apix

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/consts"
	"github.com/rs/xid"
	"net/http"
	"net/http/httptest"
	"strconv"
)

func NewCtx() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	SetTraceID(ctx)
	return ctx
}

func SetTraceID(ctx *gin.Context) {
	ctx.Set(consts.TraceIDKey, xid.New().String())
	newCtx := context.WithValue(ctx.Request.Context(), consts.TraceIDKey, ctx.GetString(consts.TraceIDKey))
	ctx.Request = ctx.Request.WithContext(newCtx)
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
	newCtx := context.WithValue(ctx.Request.Context(), consts.UserIDKey, userID)
	ctx.Request = ctx.Request.WithContext(newCtx)
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
