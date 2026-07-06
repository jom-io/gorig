package tokenx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/global/consts"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
)

const (
	redisTokenPrefix = "gorig:tokenx:"
	redisScanCount   = 100
)

type redisImpl struct {
	generator TokenGenerator
	redis     *cache.RedisCache[tokenInfo]
}

func newRedisImpl(generator TokenGenerator) *redisImpl {
	return &redisImpl{
		generator: generator,
		redis:     cache.GetRedisInstance[tokenInfo](context.Background()),
	}
}

func redisTokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return redisTokenPrefix + "token:" + hex.EncodeToString(sum[:])
}

func redisUserTokensKey(userID string) string {
	return redisTokenPrefix + "user:" + userID + ":tokens"
}

func redisUsersKey() string {
	return redisTokenPrefix + "users"
}

func (u *redisImpl) ready() bool {
	return u != nil && u.redis != nil && u.redis.IsInitialized()
}

func (u *redisImpl) ctx() context.Context {
	if !u.ready() || u.redis.GetCtx() == nil {
		return context.Background()
	}
	return u.redis.GetCtx()
}

func tokenTTL(expireAt int64) time.Duration {
	return time.Until(time.Unix(expireAt, 0))
}

func (u *redisImpl) getTokenInfo(token string) (*tokenInfo, bool) {
	if !u.ready() {
		return nil, false
	}
	info, err := u.redis.Get(redisTokenKey(token))
	if err != nil {
		return nil, false
	}
	if info.ExpiresAt < time.Now().Unix() {
		u.removeToken(token, &info)
		return nil, false
	}
	return &info, true
}

func (u *redisImpl) storeToken(token string, info *tokenInfo) bool {
	if !u.ready() || info == nil || token == "" || info.UserID == "" {
		return false
	}

	ttl := tokenTTL(info.ExpiresAt)
	if ttl <= 0 {
		return false
	}

	if err := u.redis.Set(redisTokenKey(token), *info, ttl); err != nil {
		logger.Error(nil, fmt.Sprintf("redis token set error:%v", err))
		return false
	}

	ctx := u.ctx()
	userKey := redisUserTokensKey(info.UserID)
	pipe := u.redis.Client.TxPipeline()
	pipe.SAdd(ctx, userKey, token)
	pipe.SAdd(ctx, redisUsersKey(), info.UserID)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Error(nil, fmt.Sprintf("redis token index set error:%v", err))
		return false
	}
	u.extendUserTokenIndexTTL(userKey, ttl)
	return true
}

func (u *redisImpl) extendUserTokenIndexTTL(userKey string, ttl time.Duration) {
	if !u.ready() || userKey == "" || ttl <= 0 {
		return
	}

	ctx := u.ctx()
	currentTTL, err := u.redis.Client.TTL(ctx, userKey).Result()
	if err != nil {
		logger.Error(nil, fmt.Sprintf("redis user token ttl error:%v", err))
		return
	}
	if currentTTL < ttl {
		if err := u.redis.Client.Expire(ctx, userKey, ttl).Err(); err != nil {
			logger.Error(nil, fmt.Sprintf("redis user token expire error:%v", err))
		}
	}
}

func (u *redisImpl) removeToken(token string, info *tokenInfo) {
	if !u.ready() || token == "" {
		return
	}

	if info == nil {
		if stored, ok := u.getTokenInfo(token); ok {
			info = stored
		} else if claims, err := u.generator.Parse(token); err == nil {
			info = &tokenInfo{UserID: claims.UserId}
		}
	}

	ctx := u.ctx()
	pipe := u.redis.Client.TxPipeline()
	pipe.Del(ctx, redisTokenKey(token))
	if info != nil && info.UserID != "" {
		userKey := redisUserTokensKey(info.UserID)
		pipe.SRem(ctx, userKey, token)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Error(nil, fmt.Sprintf("redis token remove error:%v", err))
	}

	if info != nil && info.UserID != "" {
		userKey := redisUserTokensKey(info.UserID)
		n, err := u.redis.Client.SCard(ctx, userKey).Result()
		if err == nil && n == 0 {
			_ = u.redis.Client.Del(ctx, userKey).Err()
			_ = u.redis.Client.SRem(ctx, redisUsersKey(), info.UserID).Err()
		}
	}
}

func (u *redisImpl) findUserToken(userID, userType string) string {
	if !u.ready() || userID == "" {
		return ""
	}

	tokens, err := u.redis.Client.SMembers(u.ctx(), redisUserTokensKey(userID)).Result()
	if err != nil {
		logger.Error(nil, fmt.Sprintf("redis user token members error:%v", err))
		return ""
	}

	for _, token := range tokens {
		info, ok := u.getTokenInfo(token)
		if !ok {
			u.removeToken(token, nil)
			continue
		}
		if info.UserID == userID && info.UserType == userType {
			return token
		}
	}
	return ""
}

func (u *redisImpl) Record(userToken string, userInfo map[string]interface{}) bool {
	if customClaims, err := u.generator.Parse(userToken); err == nil {
		expireAt := time.Now().Unix() + int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
		return u.storeToken(userToken, &tokenInfo{
			UserID:    customClaims.UserId,
			UserType:  getUserType(userInfo),
			ExpiresAt: expireAt,
		})
	}
	return false
}

func (u *redisImpl) GenerateAndRecord(ctx context.Context, userID string, userInfo map[string]interface{}, expireAt int64) (token string, err *errors.Error) {
	logger.Info(ctx, fmt.Sprintf("GenerateAndRecord userId:%s userInfo:%v expireAt:%d", userID, userInfo, expireAt))
	if !u.ready() {
		return "", errors.Sys("redis token manager is not initialized")
	}

	if expireAt < time.Now().Unix() {
		expireAt = time.Now().Unix() + int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
	}

	if token = u.findUserToken(userID, getUserType(userInfo)); token != "" {
		return token, nil
	}

	if token, err = u.generator.Generate(userID, userInfo, expireAt); err == nil {
		u.Clean(userID)
		if !u.Record(token, userInfo) {
			return "", errors.Sys("redis token record failed")
		}
	}
	return
}

func (u *redisImpl) IsNotExpired(token string, expireAtSec int64) (*CustomClaims, int) {
	if customClaims, err := u.generator.Parse(token); err == nil {
		if time.Now().Unix()-(customClaims.ExpiresAt+expireAtSec) < 0 {
			return customClaims, consts.JwtTokenOK
		}
		return customClaims, consts.JwtTokenExpired
	}
	return nil, consts.JwtTokenInvalid
}

func (u *redisImpl) IsMeetRefresh(token string) bool {
	_, code := u.IsNotExpired(token, int64(configure.GetInt("Jwt.TokenRefreshAllowSec")))
	switch code {
	case consts.JwtTokenOK, consts.JwtTokenExpired:
		return true
	}
	return false
}

func (u *redisImpl) Refresh(oldToken string, newToken string) bool {
	if !u.ready() {
		return false
	}

	customClaims, err := u.generator.Parse(oldToken)
	if err != nil {
		return false
	}

	info, ok := u.getTokenInfo(oldToken)
	if !ok {
		info = &tokenInfo{
			UserID:   customClaims.UserId,
			UserType: getUserType(customClaims.UserInfo),
		}
	}
	info.ExpiresAt = time.Now().Unix() + int64(configure.GetInt("Jwt.TokenRefreshExpireAt", defExpire))
	info.LastRefresh = time.Now().Unix()

	u.removeToken(oldToken, info)
	return u.storeToken(newToken, info)
}

func (u *redisImpl) IsEffective(token string) bool {
	_, code := u.IsNotExpired(token, 0)
	if code != consts.JwtTokenOK {
		return false
	}
	_, ok := u.GetUserID(token)
	return ok
}

func (u *redisImpl) Destroy(token string) {
	logger.Info(nil, fmt.Sprintf("Destroy token:%s", token))
	u.removeToken(token, nil)
}

func (u *redisImpl) GetUserID(token string) (string, bool) {
	info, ok := u.getTokenInfo(token)
	if !ok {
		return "", false
	}

	now := time.Now().Unix()
	if info.LastRefresh == 0 {
		info.LastRefresh = info.ExpiresAt - int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
	}
	if now-info.LastRefresh >= refreshGap {
		info.ExpiresAt = now + int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
		info.LastRefresh = now
		if !u.storeToken(token, info) {
			return "", false
		}
	}

	return info.UserID, true
}

func (u *redisImpl) CleanAll() {
	if !u.ready() {
		return
	}

	ctx := u.ctx()
	var cursor uint64
	for {
		keys, next, err := u.redis.Client.Scan(ctx, cursor, redisTokenPrefix+"*", redisScanCount).Result()
		if err != nil {
			logger.Error(nil, fmt.Sprintf("redis token scan error:%v", err))
			return
		}
		if len(keys) > 0 {
			if err := u.redis.Client.Del(ctx, keys...).Err(); err != nil {
				logger.Error(nil, fmt.Sprintf("redis token clean all error:%v", err))
				return
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func (u *redisImpl) Clean(userID string) {
	if !u.ready() || userID == "" {
		return
	}

	ctx := u.ctx()
	userKey := redisUserTokensKey(userID)
	tokens, err := u.redis.Client.SMembers(ctx, userKey).Result()
	if err != nil {
		logger.Error(nil, fmt.Sprintf("redis user token clean members error:%v", err))
		return
	}

	if len(tokens) > 0 {
		keys := make([]string, 0, len(tokens))
		for _, token := range tokens {
			keys = append(keys, redisTokenKey(token))
		}
		if err := u.redis.Client.Del(ctx, keys...).Err(); err != nil {
			logger.Error(nil, fmt.Sprintf("redis user token clean error:%v", err))
			return
		}
	}

	pipe := u.redis.Client.TxPipeline()
	pipe.Del(ctx, userKey)
	pipe.SRem(ctx, redisUsersKey(), userID)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Error(nil, fmt.Sprintf("redis user token index clean error:%v", err))
	}
}
