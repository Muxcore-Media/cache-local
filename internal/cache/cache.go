package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultTTL    = 5 * time.Minute
	sweepInterval = 1 * time.Minute
)

type entry struct {
	data      []byte
	expiresAt time.Time
}

// Cache is an in-memory CacheLayer with TTL expiry and a background sweeper.
type Cache struct {
	mu        sync.RWMutex
	entries   map[string]*entry
	closed    chan struct{}
	closeOnce sync.Once
	ttl       time.Duration
}

// New creates an in-memory cache with the default TTL and starts the sweeper.
func New() *Cache {
	return NewWithTTL(defaultTTL)
}

// NewWithTTL creates a cache with a custom default TTL (used by tests).
func NewWithTTL(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	c := &Cache{
		entries: make(map[string]*entry),
		closed:  make(chan struct{}),
		ttl:     ttl,
	}
	go c.sweepLoop()
	return c
}

func (c *Cache) sweepLoop() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.closed:
			return
		case <-ticker.C:
			c.sweep()
		}
	}
}

func (c *Cache) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}

// Close stops the background sweeper. Does not clear entries.
func (c *Cache) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
}

func (c *Cache) Get(_ context.Context, key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}
	return e.data, true
}

func (c *Cache) Set(_ context.Context, key string, data []byte) error {
	if key == "" {
		return fmt.Errorf("cache: key must not be empty")
	}
	c.mu.Lock()
	c.entries[key] = &entry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
	return nil
}

func (c *Cache) Invalidate(_ context.Context, prefix string) error {
	if prefix == "" {
		return nil
	}
	c.mu.Lock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
	return nil
}
