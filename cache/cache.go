package cache

import (
	"sync"
	"time"
)

// CacheEntry represents a cached value with expiration time
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// Cache provides a thread-safe cache with TTL support
type Cache struct {
	data            sync.Map
	cleanupInterval time.Duration
	stopCh          chan struct{}
	once            sync.Once
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	c := &Cache{
		cleanupInterval: 1 * time.Minute,
		stopCh:          make(chan struct{}),
	}

	go c.startCleanup()

	return c
}

// Get retrieves a value from the cache if it exists and hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	value, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}

	entry, ok := value.(CacheEntry)
	if !ok {
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		c.data.Delete(key)
		return nil, false
	}

	return entry.Data, true
}

// Set stores a value in the cache with the specified TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	entry := CacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}
	c.data.Store(key, entry)
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.data.Delete(key)
}

// Clear removes all values from the cache
func (c *Cache) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})
}

// startCleanup runs a background goroutine that periodically removes expired entries
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCh:
			return
		}
	}
}

// cleanupExpired removes all expired entries from the cache
func (c *Cache) cleanupExpired() {
	now := time.Now()
	c.data.Range(func(key, value interface{}) bool {
		entry, ok := value.(CacheEntry)
		if !ok {
			return true
		}
		if now.After(entry.ExpiresAt) {
			c.data.Delete(key)
		}
		return true
	})
}

// Close stops the background cleanup goroutine
func (c *Cache) Close() {
	c.once.Do(func() {
		close(c.stopCh)
	})
}
