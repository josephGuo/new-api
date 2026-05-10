package common

import (
	"fmt"
	"hash/fnv"
	"runtime"
	"sync"
	"time"
)

// ShardedInMemoryRateLimiter 分片内存限流器
// 使用分片锁减少高并发下的锁竞争
type ShardedInMemoryRateLimiter struct {
	shards    []*shard
	numShards int
	hash      func(string) int
}

// shard 单个分片，包含独立的锁和存储
type shard struct {
	store              map[string]*[]int64
	mutex              sync.Mutex
	expirationDuration time.Duration
}

// NewShardedRateLimiter 创建分片限流器
// numShards: 分片数量，建议为CPU核心数的2-4倍
// expirationDuration: 过期清理间隔
func NewShardedRateLimiter(numShards int, expirationDuration time.Duration) *ShardedInMemoryRateLimiter {
	if numShards <= 0 {
		numShards = 32 // 默认32个分片
	}

	limiter := &ShardedInMemoryRateLimiter{
		shards:    make([]*shard, numShards),
		numShards: numShards,
		hash:      fnvHash,
	}

	for i := 0; i < numShards; i++ {
		limiter.shards[i] = &shard{
			store:              make(map[string]*[]int64),
			expirationDuration: expirationDuration,
		}

		// 每个分片独立启动清理协程
		if expirationDuration > 0 {
			go limiter.shards[i].clearExpiredItems()
		}
	}

	return limiter
}

// fnvHash 使用FNV哈希算法计算key的哈希值
func fnvHash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32())
}

// getShard 根据key获取对应的分片
func (s *ShardedInMemoryRateLimiter) getShard(key string) *shard {
	index := s.hash(key) % s.numShards
	if index < 0 {
		index = -index
	}
	return s.shards[index]
}

// Request 检查是否允许请求（duration单位：秒）
func (s *ShardedInMemoryRateLimiter) Request(key string, maxRequestNum int, duration int64) bool {
	shard := s.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	// [old <-- new]
	queue, ok := shard.store[key]
	now := time.Now().Unix()

	if ok {
		if len(*queue) < maxRequestNum {
			*queue = append(*queue, now)
			return true
		} else {
			// 检查最旧的请求是否已过期
			if now-(*queue)[0] >= duration {
				*queue = (*queue)[1:]
				*queue = append(*queue, now)
				return true
			} else {
				return false
			}
		}
	} else {
		// 预分配容量，减少扩容
		q := make([]int64, 0, maxRequestNum)
		q = append(q, now)
		shard.store[key] = &q
	}
	return true
}

// clearExpiredItems 清理过期的限流记录
func (s *shard) clearExpiredItems() {
	for {
		time.Sleep(s.expirationDuration)

		s.mutex.Lock()
		now := time.Now().Unix()
		expirationSeconds := int64(s.expirationDuration.Seconds())

		for key := range s.store {
			queue := s.store[key]
			size := len(*queue)
			if size == 0 || now-(*queue)[size-1] > expirationSeconds {
				delete(s.store, key)
			}
		}
		s.mutex.Unlock()
	}
}

// GetStats 获取统计信息（用于监控）
func (s *ShardedInMemoryRateLimiter) GetStats() map[string]int {
	stats := make(map[string]int)
	totalKeys := 0

	for i, shard := range s.shards {
		shard.mutex.Lock()
		count := len(shard.store)
		shard.mutex.Unlock()

		stats[shardName(i)] = count
		totalKeys += count
	}

	stats["total"] = totalKeys
	return stats
}

func shardName(index int) string {
	return fmt.Sprintf("shard_%d", index)
}

// 全局分片限流器实例（向后兼容）
var shardedRateLimiter *ShardedInMemoryRateLimiter

// InitShardedRateLimiter 初始化全局分片限流器
func InitShardedRateLimiter(expirationDuration time.Duration) {
	if shardedRateLimiter == nil {
		// 使用CPU核心数的4倍作为分片数，最少32个分片
		numShards := runtime.NumCPU() * 4
		if numShards < 32 {
			numShards = 32
		}
		shardedRateLimiter = NewShardedRateLimiter(numShards, expirationDuration)
	}
}

// ShardedRateLimitRequest 使用分片限流器检查请求
func ShardedRateLimitRequest(key string, maxRequestNum int, duration int64) bool {
	if shardedRateLimiter == nil {
		// 降级到简单实现
		return false
	}
	return shardedRateLimiter.Request(key, maxRequestNum, duration)
}
