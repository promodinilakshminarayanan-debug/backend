// Package cache provides a minimal thread-safe, in-memory key/value
// store with per-entry TTL expiry — used to avoid hitting the upstream
// exchange-rate API on every request.
package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     map[string]float64
	expiresAt time.Time
}

// Cache is safe for concurrent use by multiple goroutines.
type Cache struct {
	mu   sync.RWMutex
	data map[string]entry
	ttl  time.Duration
}

// New creates a Cache whose entries expire ttl after they're Set.
func New(ttl time.Duration) *Cache {
	return &Cache{
		data: make(map[string]entry),
		ttl:  ttl,
	}
}

// Get returns the cached value for key and whether it was found and
// still fresh. An expired entry is treated as a miss.
func (c *Cache) Get(key string) (map[string]float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.data[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Set stores value under key, resetting its TTL.
func (c *Cache) Set(key string, value map[string]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}
