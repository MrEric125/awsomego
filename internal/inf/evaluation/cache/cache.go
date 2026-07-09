package cache

import (
	"awesome/internal/inf/evaluation/logger"
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// MemoryCache 内存缓存
type MemoryCache struct {
	items   map[string]*CacheItem
	mu      sync.RWMutex
	logger  *logger.Logger
	maxSize int
}

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int) *MemoryCache {
	cache := &MemoryCache{
		items:   make(map[string]*CacheItem),
		logger:  logger.GetLogger().WithComponent("cache"),
		maxSize: maxSize,
	}

	// 启动清理协程
	go cache.cleanupRoutine()

	return cache
}

// Get 获取缓存
func (c *MemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.Expiration) {
		return nil, false
	}

	return item.Value, true
}

// Set 设置缓存
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查缓存大小
	if len(c.items) >= c.maxSize {
		// 删除最旧的项
		c.evictOldest()
	}

	c.items[key] = &CacheItem{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}

	c.logger.Debug("Cache item set",
		zap.String("key", key),
		zap.Duration("ttl", ttl))

	return nil
}

// Delete 删除缓存
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)

	c.logger.Debug("Cache item deleted",
		zap.String("key", key))

	return nil
}

// Clear 清空缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)

	c.logger.Info("Cache cleared")

	return nil
}

// evictOldest 驱逐最旧的项
func (c *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.Expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Expiration
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.logger.Debug("Evicted oldest cache item",
			zap.String("key", oldestKey))
	}
}

// cleanupRoutine 清理例程
func (c *MemoryCache) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

// cleanupExpired 清理过期项
func (c *MemoryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	for key, item := range c.items {
		if now.After(item.Expiration) {
			delete(c.items, key)
			count++
		}
	}

	if count > 0 {
		c.logger.Debug("Cleaned up expired cache items",
			zap.Int("count", count))
	}
}

// GetStatistics 获取统计信息
func (c *MemoryCache) GetStatistics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"total_items": len(c.items),
		"max_size":    c.maxSize,
	}
}

// MultiLevelCache 多级缓存
type MultiLevelCache struct {
	levels []Cache
	logger *logger.Logger
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(levels ...Cache) *MultiLevelCache {
	return &MultiLevelCache{
		levels: levels,
		logger: logger.GetLogger().WithComponent("multi_level_cache"),
	}
}

// Get 获取缓存（从L1到Ln依次查找）
func (c *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool) {
	for i, cache := range c.levels {
		if value, found := cache.Get(ctx, key); found {
			// 回填到更高级缓存
			if i > 0 {
				go func(level int) {
					// 使用默认TTL回填
					_ = c.levels[level-1].Set(ctx, key, value, 5*time.Minute)
				}(i)
			}
			return value, true
		}
	}
	return nil, false
}

// Set 设置缓存（写入所有级别）
func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	for _, cache := range c.levels {
		if err := cache.Set(ctx, key, value, ttl); err != nil {
			c.logger.Error("Failed to set cache",
				zap.String("key", key),
				zap.Error(err))
			// 继续设置其他级别
		}
	}
	return nil
}

// Delete 删除缓存（从所有级别删除）
func (c *MultiLevelCache) Delete(ctx context.Context, key string) error {
	for _, cache := range c.levels {
		if err := cache.Delete(ctx, key); err != nil {
			c.logger.Error("Failed to delete cache",
				zap.String("key", key),
				zap.Error(err))
		}
	}
	return nil
}

// Clear 清空缓存（清空所有级别）
func (c *MultiLevelCache) Clear(ctx context.Context) error {
	for _, cache := range c.levels {
		if err := cache.Clear(ctx); err != nil {
			c.logger.Error("Failed to clear cache", zap.Error(err))
		}
	}
	return nil
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
	StartTime time.Time
}

// HitRate 计算命中率
func (s *CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}
