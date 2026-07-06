package test

import (
	"context"
	"testing"
	"time"

	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/mid/tokenx"
)

func redisTokenServiceOrSkip(t *testing.T) *tokenx.TokenService {
	t.Helper()

	redis := cache.GetRedisInstance[string](context.Background())
	if redis == nil || !redis.IsInitialized() {
		t.Skip("redis is not configured")
	}

	svc := tokenx.Get(tokenx.Jwt, tokenx.Redis)
	svc.Manager.CleanAll()
	t.Cleanup(func() {
		svc.Manager.CleanAll()
	})
	return svc
}

func TestRedisTokenManager_GenerateReuseAndClean(t *testing.T) {
	svc := redisTokenServiceOrSkip(t)
	ctx := context.Background()

	userID := "redis-token-user-1"
	userInfo := map[string]interface{}{"role": "admin"}

	token1, err := svc.Manager.GenerateAndRecord(ctx, userID, userInfo, 0)
	if err != nil {
		t.Fatalf("GenerateAndRecord failed: %v", err)
	}
	if token1 == "" {
		t.Fatal("expected token, got empty string")
	}

	gotUserID, ok := svc.Manager.GetUserID(token1)
	if !ok || gotUserID != userID {
		t.Fatalf("GetUserID expected %s true, got %s %v", userID, gotUserID, ok)
	}

	token2, err := svc.Manager.GenerateAndRecord(ctx, userID, userInfo, 0)
	if err != nil {
		t.Fatalf("GenerateAndRecord reuse failed: %v", err)
	}
	if token2 != token1 {
		t.Fatalf("expected same token for same user info")
	}

	token3, err := svc.Manager.GenerateAndRecord(ctx, userID, map[string]interface{}{"role": "operator"}, 0)
	if err != nil {
		t.Fatalf("GenerateAndRecord changed userInfo failed: %v", err)
	}
	if token3 == "" || token3 == token1 {
		t.Fatalf("expected new token when user info changes")
	}
	if _, ok := svc.Manager.GetUserID(token1); ok {
		t.Fatal("expected old token to be cleaned")
	}

	svc.Manager.Clean(userID)
	if _, ok := svc.Manager.GetUserID(token3); ok {
		t.Fatal("expected token to be cleaned by user id")
	}
}

func TestRedisTokenManager_RefreshDestroyAndEffective(t *testing.T) {
	svc := redisTokenServiceOrSkip(t)
	ctx := context.Background()

	userID := "redis-token-user-2"
	userInfo := map[string]interface{}{"role": "member"}

	oldToken, err := svc.Manager.GenerateAndRecord(ctx, userID, userInfo, 0)
	if err != nil {
		t.Fatalf("GenerateAndRecord failed: %v", err)
	}

	time.Sleep(time.Second)
	newToken, err := svc.Generator.Generate(userID, userInfo, 60)
	if err != nil {
		t.Fatalf("Generate new token failed: %v", err)
	}
	if newToken == oldToken {
		t.Fatal("expected generated refresh token to differ")
	}

	if !svc.Manager.Refresh(oldToken, newToken) {
		t.Fatal("Refresh expected true")
	}
	if _, ok := svc.Manager.GetUserID(oldToken); ok {
		t.Fatal("expected old token to be removed after refresh")
	}

	gotUserID, ok := svc.Manager.GetUserID(newToken)
	if !ok || gotUserID != userID {
		t.Fatalf("new token GetUserID expected %s true, got %s %v", userID, gotUserID, ok)
	}
	if !svc.Manager.IsEffective(newToken) {
		t.Fatal("expected new token to be effective")
	}

	svc.Manager.Destroy(newToken)
	if _, ok := svc.Manager.GetUserID(newToken); ok {
		t.Fatal("expected destroyed token to be invalid")
	}
	if svc.Manager.IsEffective(newToken) {
		t.Fatal("expected destroyed token to be ineffective")
	}
}
