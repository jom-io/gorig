package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
)

// Deprecated: Use apix.NewCtx() instead.
func NewCtx() *gin.Context {
	return apix.NewCtx()
}

// Deprecated: Use apix.SetTraceID() instead.
func SetTraceID(ctx *gin.Context) {
	apix.SetTraceID(ctx)
}

// Deprecated: Use apix.GetTraceID() instead.
func GetTraceID(ctx *gin.Context) string {
	return apix.GetTraceID(ctx)
}

// Deprecated: Use apix.GetUserID() instead.
func GetUserID(ctx *gin.Context) string {
	return apix.GetUserID(ctx)
}

// Deprecated: Use apix.GetUserIDInt() instead.
func GetUserIDInt(ctx *gin.Context) int {
	return apix.GetUserIDInt(ctx)
}

// Deprecated: Use apix.GetUserIDInt64() instead.
func GetUserIDInt64(ctx *gin.Context) int64 {
	return apix.GetUserIDInt64(ctx)
}

// Deprecated: Use apix.SetUserID() instead.
func SetUserID(ctx *gin.Context, userID string) {
	apix.SetUserID(ctx, userID)
}

// Deprecated: Use apix.GetUserInfo() instead.
func GetUserInfo(ctx *gin.Context) map[string]interface{} {
	return apix.GetUserInfo(ctx)
}

// Deprecated: Use apix.SetUserInfo() instead.
func SetUserInfo(ctx *gin.Context, userInfo map[string]interface{}) {
	apix.SetUserInfo(ctx, userInfo)
}

// Deprecated: Use apix.GetUserInfoValue() instead.
func GetUserInfoValue(ctx *gin.Context, key string) any {
	return apix.GetUserInfoValue(ctx, key)
}

// Deprecated: Use apix.GetClientIP() instead.
func GetClientIP(ctx *gin.Context) string {
	return apix.GetClientIP(ctx)
}

func DumpRouters(call func(info gin.RouteInfo)) {
	routes := gEngine.Routes()
	for _, route := range routes {
		call(route)
	}
}
