package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"vmmanager/config"
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	HealthCheck(ctx context.Context) error
	Close() error
	SetSession(ctx context.Context, sessionID string, data interface{}, expiration time.Duration) error
	GetSession(ctx context.Context, sessionID string, dest interface{}) error
	DeleteSession(ctx context.Context, sessionID string) error
	SetToken(ctx context.Context, tokenID string, data interface{}, expiration time.Duration) error
	GetToken(ctx context.Context, tokenID string, dest interface{}) error
	DeleteToken(ctx context.Context, tokenID string) error
	IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, error)
	GetRateLimit(ctx context.Context, key string) (int64, error)
}

type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]cacheItem
}

type cacheItem struct {
	value      []byte
	expiration time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]cacheItem),
	}
}

func NewMemoryCacheWithConfig(cfg *config.RedisConfig) *MemoryCache {
	return NewMemoryCache()
}

func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	c.mu.Lock()
	c.data[key] = cacheItem{
		value:      data,
		expiration: time.Now().Add(expiration),
	}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	if time.Now().After(item.expiration) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return fmt.Errorf("key expired: %s", key)
	}

	return json.Unmarshal(item.value, dest)
}

func (c *MemoryCache) Delete(ctx context.Context, keys ...string) error {
	c.mu.Lock()
	for _, key := range keys {
		delete(c.data, key)
	}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := int64(0)
	now := time.Now()
	for _, key := range keys {
		if item, ok := c.data[key]; ok && now.Before(item.expiration) {
			count++
		}
	}
	return count, nil
}

func (c *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return -2 * time.Second, fmt.Errorf("key not found: %s", key)
	}

	remaining := time.Until(item.expiration)
	if remaining < 0 {
		return -2 * time.Second, nil
	}
	return remaining, nil
}

func (c *MemoryCache) HealthCheck(ctx context.Context) error {
	return nil
}

func (c *MemoryCache) Close() error {
	c.mu.Lock()
	c.data = make(map[string]cacheItem)
	c.mu.Unlock()
	return nil
}

const (
	SessionPrefix   = "session:"
	TokenPrefix     = "token:"
	VMStatsPrefix   = "vm_stats:"
	TemplatePrefix  = "template:"
	UserPrefix      = "user:"
	RateLimitPrefix = "rate_limit:"
)

func (c *MemoryCache) SetSession(ctx context.Context, sessionID string, data interface{}, expiration time.Duration) error {
	return c.Set(ctx, SessionPrefix+sessionID, data, expiration)
}

func (c *MemoryCache) GetSession(ctx context.Context, sessionID string, dest interface{}) error {
	return c.Get(ctx, SessionPrefix+sessionID, dest)
}

func (c *MemoryCache) DeleteSession(ctx context.Context, sessionID string) error {
	return c.Delete(ctx, SessionPrefix+sessionID)
}

func (c *MemoryCache) SetToken(ctx context.Context, tokenID string, data interface{}, expiration time.Duration) error {
	return c.Set(ctx, TokenPrefix+tokenID, data, expiration)
}

func (c *MemoryCache) GetToken(ctx context.Context, tokenID string, dest interface{}) error {
	return c.Get(ctx, TokenPrefix+tokenID, dest)
}

func (c *MemoryCache) DeleteToken(ctx context.Context, tokenID string) error {
	return c.Delete(ctx, TokenPrefix+tokenID)
}

func (c *MemoryCache) IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := RateLimitPrefix + key
	count, _ := c.Exists(ctx, fullKey)
	count++
	c.Set(ctx, fullKey, count, window)
	return count, nil
}

func (c *MemoryCache) GetRateLimit(ctx context.Context, key string) (int64, error) {
	fullKey := RateLimitPrefix + key
	var count int64
	err := c.Get(ctx, fullKey, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

type SessionData struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RateLimitData struct {
	Count       int64     `json:"count"`
	WindowStart time.Time `json:"window_start"`
}
