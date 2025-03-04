package cache

import (
	"github.com/gin-gonic/gin"
	"time"
)

var caches []Cache[any]

var SimpleTool = func(ctx *gin.Context) *Tool[any] {
	return NewCacheTool[any](ctx, caches, nil)
}

func init() {

	l1Cache := NewGoCache[any](10*time.Minute, 5*time.Minute)

	caches = []Cache[any]{l1Cache}

	l2Cache := GetRedisInstance[any]()
	if l2Cache != nil {
		caches = append(caches, l2Cache)
	}

	// 定义一个 LoaderFunc，用于从外部源加载数据
	//loader := func(key string) (string, error) {
	//	// 在实际应用中，这里可以是数据库查询或其他外部数据源
	//	fmt.Printf("Loading data for key: %s from external source\n", key)
	//	return fmt.Sprintf("Data_for_%s", key), nil
	//}

	//// 创建 Tool
	//cacheTool := NewCacheTool[any](caches, loader)
	//
	//// 示例：设置缓存数据
	//if err := cacheTool.Set("user:123", "John Doe", 10*time.Minute); err != nil {
	//	log.Fatalf("Error setting cache: %v", err)
	//}
	//fmt.Println("Set cache for key 'user:123'")
	//
	//// 示例：获取缓存数据（命中 L1）
	//data, err := cacheTool.Get("user:123", 10*time.Minute)
	//if err != nil {
	//	log.Fatalf("Error getting cache: %v", err)
	//}
	//fmt.Printf("Retrieved cache for 'user:123': %s\n", data)
	//
	//// 示例：获取未缓存的数据（将触发 LoaderFunc）
	//data, err = cacheTool.Get("user:456", 10*time.Minute)
	//if err != nil {
	//	log.Fatalf("Error getting cache: %v", err)
	//}
	//fmt.Printf("Retrieved cache for 'user:456': %s\n", data)
	//
	//// 示例：删除缓存数据
	//if err := cacheTool.Delete("user:123"); err != nil {
	//	log.Fatalf("Error deleting cache: %v", err)
	//}
	//fmt.Println("Deleted cache for key 'user:123'")
}
