package common

/*
// 在初始化时创建分片限流器
func init() {
    common.InitShardedRateLimiter(20 * time.Minute)
}

// 在业务代码中使用
func rateLimitHandler(c *gin.Context) {
    key := c.ClientIP()
    if !common.ShardedRateLimitRequest(key, 100, 60) {
        c.Status(http.StatusTooManyRequests)
        c.Abort()
        return
    }
    c.Next()
}
需要更新 middleware/rate-limit.go 中的使用方式：
// 替换前
var inMemoryRateLimiter common.InMemoryRateLimiter

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
    key := mark + c.ClientIP()
    if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
        c.Status(http.StatusTooManyRequests)
        c.Abort()
        return
    }
}

// 替换后
func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
    key := mark + c.ClientIP()
    if !common.ShardedRateLimitRequest(key, maxRequestNum, duration) {
        c.Status(http.StatusTooManyRequests)
        c.Abort()
        return
    }
}
*/
