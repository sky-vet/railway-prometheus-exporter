package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// CacheEntry represents a cached value with expiration time
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
	AccessedAt time.Time // Track last access for LRU eviction
}

// Cache provides a thread-safe cache with TTL support and LRU eviction
type Cache struct {
	data            sync.Map
	size            atomic.Int64  // Track current size atomically
	maxSize         int64         // Maximum number of entries
	cleanupInterval time.Duration
	stopCh          chan struct{}
	once            sync.Once
}

// NewCache creates a new cache instance with default max size
func NewCache() *Cache {
	return NewCacheWithMaxSize(10000) // Default max size
}

// NewCacheWithMaxSize creates a new cache instance with specified max size
func NewCacheWithMaxSize(maxSize int) *Cache {
	c := &Cache{
		maxSize:         int64(maxSize),
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
		c.Delete(key) // Use Delete to properly decrement size
		return nil, false
	}

	// Update access time for LRU (without incrementing size)
	entry.AccessedAt = time.Now()
	c.data.Store(key, entry)

	return entry.Data, true
}

// Set stores a value in the cache with the specified TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	now := time.Now()
	entry := CacheEntry{
		Data:       value,
		ExpiresAt:  now.Add(ttl),
		AccessedAt: now,
	}

	// Check if key already exists
	_, exists := c.data.Load(key)

	c.data.Store(key, entry)

	// Only increment size if it's a new entry
	if !exists {
		newSize := c.size.Add(1)

		// Check if we need to evict entries
		if newSize > c.maxSize {
			c.evictLRU()
		}
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	if _, exists := c.data.LoadAndDelete(key); exists {
		c.size.Add(-1)
	}
}

// Clear removes all values from the cache
func (c *Cache) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})
	c.size.Store(0)
}

// Size returns the current number of entries in the cache
func (c *Cache) Size() int64 {
	return c.size.Load()
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
			if keyStr, ok := key.(string); ok {
				c.Delete(keyStr) // Use Delete to properly decrement size
			}
		}
		return true
	})
}

// evictLRU removes least recently used entries when cache is full
func (c *Cache) evictLRU() {
	// Calculate how many entries to remove (10% of cache or at least 1)
	targetSize := int64(float64(c.maxSize) * 0.9) // Remove 10% when full
	toRemove := c.size.Load() - targetSize
	if toRemove <= 0 {
		return
	}

	// Collect all entries with their access times
	type entryInfo struct {
		key        interface{}
		accessedAt time.Time
	}
	var entries []entryInfo

	c.data.Range(func(key, value interface{}) bool {
		entry, ok := value.(CacheEntry)
		if !ok {
			return true
		}
		entries = append(entries, entryInfo{
			key:        key,
			accessedAt: entry.AccessedAt,
		})
		return true
	})

	// Sort by access time (oldest first)
	// Simple bubble sort since we're only removing a small percentage
	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			if entries[j].accessedAt.After(entries[j+1].accessedAt) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	// Remove oldest entries
	removed := int64(0)
	for _, e := range entries {
		if removed >= toRemove {
			break
		}
		if keyStr, ok := e.key.(string); ok {
			c.Delete(keyStr)
			removed++
		}
	}
}

// Close stops the background cleanup goroutine
func (c *Cache) Close() {
	c.once.Do(func() {
		close(c.stopCh)
	})
}
