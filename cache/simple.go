package cache

import (
	"context"
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

	l2Cache := GetRedisInstance[any](context.Background())
	if l2Cache != nil {
		caches = append(caches, l2Cache)
	}

	_ = New[any](JSON, "dev")

	// Define a LoaderFunc for loading data from an external source
	//loader := func(key string) (string, error) {
	//	// In actual applications, this could be a database query or other external data source
	//	fmt.Printf("Loading data for key: %s from external source\n", key)
	//	return fmt.Sprintf("Data_for_%s", key), nil
	//}

	//// Create Tool
	//cacheTool := NewCacheTool[any](caches, loader)
	//
	//// Example: Set cache data
	//if err := cacheTool.Set("user:123", "John Doe", 10*time.Minute); err != nil {
	//	log.Fatalf("Error setting cache: %v", err)
	//}
	//fmt.Println("Set cache for key 'user:123'")
	//
	//// Example: Get cache data (hit L1)
	//data, err := cacheTool.Get("user:123", 10*time.Minute)
	//if err != nil {
	//	log.Fatalf("Error getting cache: %v", err)
	//}
	//fmt.Printf("Retrieved cache for 'user:123': %s\n", data)
	//
	//// Example: Get uncached data (will trigger LoaderFunc)
	//data, err = cacheTool.Get("user:456", 10*time.Minute)
	//if err != nil {
	//	log.Fatalf("Error getting cache: %v", err)
	//}
	//fmt.Printf("Retrieved cache for 'user:456': %s\n", data)
	//
	//// Example: Delete cache data
	//if err := cacheTool.Delete("user:123"); err != nil {
	//	log.Fatalf("Error deleting cache: %v", err)
	//}
	//fmt.Println("Deleted cache for key 'user:123'")
}
