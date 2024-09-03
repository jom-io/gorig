package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix/response"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"go.uber.org/zap"
	"hash/fnv"
	"sync"
	"time"
)

//var requestMap = struct {
//	sync.RWMutex
//	m map[string]time.Time
//}{m: make(map[string]time.Time)}

// NewShardedRequestMap 使用分片锁优化并发性能 分片锁原理：将数据通过哈希函数分散到多个分片中，每个分片独立加锁，减少锁的粒度，提高并发性能
var srm = NewShardedRequestMap()

const (
	shardCount           = 32
	maxShardMemory       = 2 * 1024 * 1024 // 2MB
	approximateEntrySize = 128             // 粗略估算每个键值对的大小，假设每个键值对约为128字节
	checkInterval        = 1 * time.Minute // 检查间隔时间
)

type ShardedRequestMap struct {
	shards [shardCount]*shard
}

type shard struct {
	sync.RWMutex
	m map[string]time.Time
}

var whiteList = map[string]bool{}

func DebouceAw(path string) {
	whiteList[path] = true
}

func Debounce(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果是白名单，直接跳过
		//logger.Info(c, "Debounce", zap.Any("path", c.Request.URL.Path))
		if _, ok := whiteList[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		// 如果是GET请求，使用请求路径作为key并且带上请求参数
		path := c.Request.URL.Path
		if c.Request.Method == "GET" {
			path += "?" + c.Request.URL.RawQuery
		}

		token := GetTokenByCtx(c, false)
		// 如果存在 userID
		id := GetUserIDByToken(token)

		var requestKey string
		if id != "" {
			// 如果存在userID，使用userID和请求路径组合作为key
			requestKey = path + ":id:" + id
		} else {
			//if token != "" {
			//	// 如果有用户token，使用用户token和请求路径组合作为key
			//	requestKey = path + ":token:" + token
			//} else {
			// 如果没有用户token，使用客户端IP地址和请求路径组合作为key
			clientIP := c.ClientIP()
			requestKey = path + ":ip:" + clientIP
			//}
		}

		//requestMap.Lock()
		//lastRequestTime, exists := requestMap.m[requestKey]

		lastRequestTime, exists := srm.Get(requestKey)
		since := time.Since(lastRequestTime)
		if exists && since < duration {
			//requestMap.Unlock()
			logger.Error(c, "Debounce", zap.Any("requestKey", requestKey), zap.Any("lastRequestTime", lastRequestTime), zap.Any("since", since), zap.Any("duration", duration))
			response.ErrorTooManyRequests(c)
			return
		}
		//requestMap.m[requestKey] = time.Now()
		//requestMap.Unlock()
		srm.Set(requestKey, time.Now())

		c.Next()
	}
}

func NewShardedRequestMap() *ShardedRequestMap {
	srm := &ShardedRequestMap{}
	for i := 0; i < shardCount; i++ {
		srm.shards[i] = &shard{m: make(map[string]time.Time)}
	}
	go srm.startCleanupRoutine() // 启动异步清理例程
	sys.Info(" # StartShardedCleanupRoutine")
	return srm
}

func (srm *ShardedRequestMap) getShard(key string) *shard {
	harsher := fnv.New32a()
	harsher.Write([]byte(key))
	return srm.shards[harsher.Sum32()%shardCount]
}

func (srm *ShardedRequestMap) Get(key string) (time.Time, bool) {
	shard := srm.getShard(key)
	shard.RLock()
	defer shard.RUnlock()
	val, ok := shard.m[key]
	return val, ok
}

func (srm *ShardedRequestMap) Set(key string, value time.Time) {
	shard := srm.getShard(key)
	shard.Lock()
	defer shard.Unlock()
	shard.m[key] = value
}

// 启动异步清理例程
func (srm *ShardedRequestMap) startCleanupRoutine() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for range ticker.C {
		for _, shard := range srm.shards {
			s := shard
			go func() {
				s.Lock()
				defer s.Unlock()
				//
				if float64(approximateEntrySize*len(s.m)) >= maxShardMemory {
					s.m = make(map[string]time.Time) // 清空分片
					logger.Info(nil, "startCleanupRoutine", zap.Any("size:", len(s.m)*approximateEntrySize), zap.Any("maxShardMemory", maxShardMemory))
				}
			}()
		}
	}
}
