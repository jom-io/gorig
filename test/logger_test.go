package test

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/httpx"
	"github.com/jom-io/gorig/utils/logger"
	"testing"
)

// TestLogger is a test for the logger package.
func TestLogger(t *testing.T) {
	ctx := logger.NewCtx()
	logger.Info(nil, "test nil info")
	logger.Info(ctx, "test info")
	logger.Debug(nil, "test nil debug")
	logger.Debug(ctx, "test debug")
	logger.Error(ctx, "test error")
	logger.Error(nil, "test nil error")
	logger.Warn(ctx, "test warn")
	logger.Warn(nil, "test nil warn")
	logger.DPanic(ctx, "test dpanic")
	logger.DPanic(nil, "test nil dpanic")

	ginCtx := httpx.NewCtx()
	logger.Info(ginCtx, "test gin info")

	var ginCtxNil *gin.Context
	logger.Info(ginCtxNil, "test gin nil info")

	//logger.Panic(ctx, "test panic")
	//logger.Panic(nil, "test nil panic")
	//logger.Fatal(ctx, "test fatal")
	//logger.Fatal(nil, "test nil fatal")
}
