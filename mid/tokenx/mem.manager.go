package tokenx

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"gorig/global/consts"
	configure "gorig/utils/cofigure"
	"gorig/utils/errors"
	"gorig/utils/logger"
	"gorig/utils/sys"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type tokenInfo struct {
	UserID    string
	UserType  string
	ExpiresAt int64
}

// var tokens = make(map[string]*tokenInfo)
var tokenMap = sync.Map{}
var localTokensFile = "./tokens.json"

type memoryImpl struct {
	generator TokenGenerator
}

func init() {
	sys.Info(" # Tokenx: Memory Token Manager")
	// 初始化时加载本地token
	loadLocalTokens()

	// 捕获终止信号，并在程序退出时保存token
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		logger.Info(nil, fmt.Sprintf("接收到信号:%v", sig))
		saveLocalTokens()
		os.Exit(0)
	}()

	// 定时清理过期token
	go func() {
		for {
			time.Sleep(time.Second * 60)
			tokenMap.Range(func(key, value interface{}) bool {
				userInfoGet := value.(*tokenInfo)
				if userInfoGet != nil && userInfoGet.ExpiresAt < time.Now().Unix() {
					tokenMap.Delete(key)
				}
				return true
			})
		}
	}()
}

// loadLocalTokens 从本地文件中加载tokens
func loadLocalTokens() {
	// 如果没有的话，创建一个空的文件
	if _, err := os.Stat(localTokensFile); os.IsNotExist(err) {
		file, e := os.Create(localTokensFile)
		if e != nil {
			logger.Error(nil, fmt.Sprintf("创建文件时发生错误:%v", e))
		}
		file.Close()
		return
	}
	// 打开文件
	file, err := os.Open(localTokensFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info(nil, "文件不存在，使用空的token映射")
			return
		}
		logger.Error(nil, fmt.Sprintf("打开文件时发生错误:%v", err))
		return
	}
	defer file.Close()

	// 读取文件内容
	data, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Error(nil, fmt.Sprintf("读取文件内容时发生错误:%v", err))
		return
	}
	if len(data) == 0 {
		return
	}
	//logger.Info(nil, fmt.Sprintf("加载token状态数据:%s", data))

	mapData := make(map[string]*tokenInfo)
	// 解析JSON内容
	err = json.Unmarshal(data, &mapData)
	if err != nil {
		logger.Error(nil, fmt.Sprintf("解析JSON内容时发生错误:%v", err))
		// 清除tokens文件
		err = os.Remove(localTokensFile)
		if err != nil {
			logger.Error(nil, fmt.Sprintf("删除文件时发生错误:%v", err))
		}
		logger.Info(nil, "已删除文件")
	}
	for k, v := range mapData {
		tokenMap.Store(k, v)
	}
	// 打印tokenMap
	logger.Info(nil, fmt.Sprintf("加载token状态数据:%v", len(mapData)))
}

// saveLocalTokens 将tokens保存到本地文件 原理：定义一个长度为1的channel，当有goroutine在保存文件时，其他goroutine会被阻塞
var saveLock = make(chan struct{}, 1)

func tokenLen() int {
	count := 0
	tokenMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func saveLocalTokens() {
	saveLock <- struct{}{}
	defer func() {
		<-saveLock
	}()

	// tokens换为tokenMap
	//logger.Info(nil, fmt.Sprintf("当前token数量:%d", tokenLen()))
	if tokenLen() == 0 {
		return
	}

	// 将tokenMap转换为Map
	mapData := make(map[string]*tokenInfo)
	tokenMap.Range(func(key, value interface{}) bool {
		mapData[key.(string)] = value.(*tokenInfo)
		return true
	})

	data, err := json.Marshal(mapData)
	//logger.Info(nil, fmt.Sprintf("保存token状态数据:%s", string(data)))
	if err != nil {
		logger.Error(nil, fmt.Sprintf("转换为JSON时发生错误:%v", err))
		return
	}

	// 写入文件
	err = ioutil.WriteFile(localTokensFile, data, 0644)
	if err != nil {
		logger.Error(nil, fmt.Sprintf("写入文件时发生错误:%v", err))
	}

	logger.Info(nil, fmt.Sprintf("保存token状态数据:%v", len(mapData)))
}

// 创建一个token锁 用于防止并发刷新token
var tokenLock = sync.Map{}

func getTokenLock(token string) *sync.Mutex {
	lock, exist := tokenLock.Load(token)
	if !exist {
		lock = &sync.Mutex{}
		tokenLock.Store(token, lock)
	}
	return lock.(*sync.Mutex)
}

// GetUserID
func (u *memoryImpl) GetUserID(token string) (userID string, exisit bool) {
	// 如果有panic，打印
	defer func() {
		if err := recover(); err != nil {
			logger.Error(nil, fmt.Sprintf("获取用户ID时发生错误:%v", err))
		}
	}()
	lock := getTokenLock(token)
	lock.Lock()
	defer lock.Unlock()
	defer func() {
		tokenLock.Delete(token)
	}()

	// 如果tokenMap为空，加载本地token
	if tokenLen() == 0 {
		loadLocalTokens()
	}
	value, exisitGet := tokenMap.Load(token)
	if exisitGet {
		userInfo := value.(*tokenInfo)
		// 判断token是否过期
		if userInfo != nil && userInfo.ExpiresAt < time.Now().Unix() {
			// 过期删除
			u.Destroy(token)
			return "", false
		}
		// 刷新过期时间 间隔时间小于1秒不刷新 防止并发刷新
		if time.Now().Unix()+int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))-userInfo.ExpiresAt <= 1 {
			return userInfo.UserID, true
		}
		//logger.Info(nil, fmt.Sprintf("刷新token过期时间:%v", userInfo))
		if userInfo != nil {
			userInfo.ExpiresAt = time.Now().Unix() + int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
			tokenMap.Store(token, userInfo)
			return userInfo.UserID, true
		}
	}
	return "", false
}

// 根据userInfo获取userType 规则为: userInfo的value拼接 用于token变动后及时失效
func getUserType(userInfo map[string]interface{}) string {
	userType := ""
	for _, v := range userInfo {
		userType += cast.ToString(v)
	}
	return userType
}

func (u *memoryImpl) GenerateAndRecord(userId string, userInfo map[string]interface{}, expireAt int64) (token string, err *errors.Error) {
	// 如果userID和同类型的userInfo已经存在，那么直接返回
	logger.Info(nil, fmt.Sprintf("GenerateAndRecord userId:%s userInfo:%v expireAt:%d", userId, userInfo, expireAt))
	if expireAt < time.Now().Unix() {
		expireAt = time.Now().Unix() + int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
	}
	tokenMap.Range(func(key, value interface{}) bool {
		userInfoGet := value.(*tokenInfo)
		if userInfoGet != nil && userInfoGet.UserID == userId && userInfoGet.UserType == getUserType(userInfo) {
			token = key.(string)
			return false
		}
		return true
	})
	if token != "" {
		return
	}
	if token, err = u.generator.Generate(userId, userInfo, expireAt); err == nil {
		u.Clean(userId)
		u.Record(token, userInfo)
	}
	return
}

func (u *memoryImpl) Record(userToken string, userInfo map[string]interface{}) bool {
	if customClaims, err := u.generator.Parse(userToken); err == nil {
		userId := customClaims.UserId
		//expiresAt := customClaims.ExpiresAt
		// 将token存储到全局变量中 variable.Tokens中 如果有redis则存储到redis中
		expireAt := time.Now().Unix() + int64(configure.GetInt("Jwt.TokenExpireAt", defExpire))
		tokenMap.Store(userToken, &tokenInfo{UserID: userId, UserType: getUserType(userInfo), ExpiresAt: expireAt})
		//logger.Info(nil, fmt.Sprintf("Record userToken:%s userId:%s userInfo:%v expireAt:%d", userToken, userId, userInfo, expireAt))
		return true
	} else {
		return false
	}
}

// IsMeetRefresh 检查token是否满足刷新条件
func (u *memoryImpl) IsMeetRefresh(token string) bool {
	// token基本信息是否有效：1.过期时间在允许的过期范围内;2.基本格式正确
	_, code := u.IsNotExpired(token, int64(configure.GetInt("Jwt.TokenRefreshAllowSec")))
	switch code {
	case consts.JwtTokenOK, consts.JwtTokenExpired:
		return true
		//在数据库的存储信息是否也符合过期刷新刷新条件
		//if model.CreateUserFactory("").OauthRefreshConditionCheck(customClaims.UserId, token) {
		//	return true
		//}
	}
	return false
}

// Refresh 刷新token的有效期（默认+3600秒，参见常量配置项）
func (u *memoryImpl) Refresh(oldToken string, newToken string) (res bool) {
	//如果token是有效的、或者在过期时间内，那么执行更新，换取新token
	if customClaims, err := u.generator.Parse(oldToken); err == nil {
		customClaims.ExpiresAt = time.Now().Unix() + int64(configure.GetInt("Jwt.TokenRefreshExpireAt", defExpire))
		userId := customClaims.UserId
		//expiresAt := customClaims.ExpiresAt
		//if model.CreateUserFactory("").OauthRefreshToken(userId, expiresAt, oldToken, newToken, clientIp) {
		//	return newToken, true
		//}
		// 将旧token删除
		//delete(tokens, oldToken)
		tokenMap.Delete(oldToken)
		// 将新token存储到全局变量中 variable.Tokens中 如果有redis则存储到redis中
		tokenMap.Store(newToken, &tokenInfo{UserID: userId, UserType: getUserType(customClaims.UserInfo), ExpiresAt: customClaims.ExpiresAt})
		return true
	}
	return false
}

// IsNotExpired 判断token本身是否未过期
// 参数解释：
// token： 待处理的token值
// expireAtSec： 过期时间延长的秒数，主要用于用户刷新token时，判断是否在延长的时间范围内，非刷新逻辑默认为0
func (u *memoryImpl) IsNotExpired(token string, expireAtSec int64) (*CustomClaims, int) {
	if customClaims, err := u.generator.Parse(token); err == nil {

		if time.Now().Unix()-(customClaims.ExpiresAt+expireAtSec) < 0 {
			// token有效
			return customClaims, consts.JwtTokenOK
		} else {
			// 过期的token
			return customClaims, consts.JwtTokenExpired
		}
	} else {
		// 无效的token
		return nil, consts.JwtTokenInvalid
	}
}

// IsEffective 判断token是否有效（未过期+数据库用户信息正常）
func (u *memoryImpl) IsEffective(token string) bool {
	_, code := u.IsNotExpired(token, 0)
	if consts.JwtTokenOK == code {
		////1.首先在redis检测是否存在某个用户对应的有效token，如果存在就直接返回，不再继续查询mysql，否则最后查询mysql逻辑，确保万无一失
		//if variable.ConfigYml.GetInt("Token.IsCacheToRedis") == 1 {
		//	tokenRedisFact := token_cache_redis.CreateUsersTokenCacheFactory(customClaims.UserId)
		//	if tokenRedisFact != nil {
		//		defer tokenRedisFact.ReleaseRedisConn()
		//		if tokenRedisFact.TokenCacheIsExists(token) {
		//			return true
		//		}
		//	}
		//}
		////2.token符合token本身的规则以后，继续在数据库校验是不是符合本系统其他设置，例如：一个用户默认只允许10个账号同时在线（10个token同时有效）
		//if model.CreateUserFactory("").OauthCheckTokenIsOk(customClaims.UserId, token) {
		//	return true
		//}
	}
	return false
}

// Destroy 销毁token，基本用不到，因为一个网站的用户退出都是直接关闭浏览器窗口，极少有户会点击“注销、退出”等按钮，销毁token其实无多大意义
func (u *memoryImpl) Destroy(token string) {
	logger.Info(nil, fmt.Sprintf("Destroy token:%s", token))
	tokenMap.Delete(token)
}

// CleanAll 清除所有token
func (u *memoryImpl) CleanAll() {
	tokenMap = sync.Map{}
}

// Clean 清除某个用户的所有token
func (u *memoryImpl) Clean(userId string) {
	tokenMap.Range(func(key, value interface{}) bool {
		userInfoGet := value.(*tokenInfo)
		if userInfoGet != nil && userInfoGet.UserID == userId {
			tokenMap.Delete(key)
		}
		return true
	})
}
