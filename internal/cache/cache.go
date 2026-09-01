package cache

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	defaultTTL           = 5 * time.Minute
	sweepInterval        = 1 * time.Minute
	DefaultMaxEntryBytes = 32 << 20 // 32 MiB — matches core default gRPC message size
	DefaultMaxBytes      = 256 << 20
)

var (
	ErrEmptyKey      = errors.New("cache: key must not be empty")
	ErrEntryTooLarge = errors.New("cache: entry exceeds max entry size")
	ErrCacheFull     = errors.New("cache: total size limit exceeded")
	ErrTooManyKeys   = errors.New("cache: entry count limit exceeded")
)

type entry struct {
	data      []byte
	expiresAt time.Time
}

// Config holds in-memory cache limits and default TTL.
type Config struct {
	TTL           time.Duration
	MaxBytes      int64
	MaxEntryBytes int64
	MaxEntries    int
	SweepInterval time.Duration
}

// Cache is an in-memory CacheLayer with TTL expiry and a background sweeper.
type Cache struct {
	mu            sync.RWMutex
	entries       map[string]*entry
	closed        chan struct{}
	closeOnce     sync.Once
	ttl           time.Duration
	maxBytes      int64
	maxEntryBytes int64
	maxEntries    int
	totalBytes    int64
	sweepInterval time.Duration
}

// New creates an in-memory cache with default limits and starts the sweeper.
func New() *Cache {
	return NewWithConfig(Config{TTL: defaultTTL})
}

// NewWithTTL creates a cache with a custom default TTL (used by tests).
func NewWithTTL(ttl time.Duration) *Cache {
	return NewWithConfig(Config{TTL: ttl})
}

// NewWithConfig creates a cache with explicit limits.
func NewWithConfig(cfg Config) *Cache {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultTTL
	}
	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	maxEntryBytes := cfg.MaxEntryBytes
	if maxEntryBytes <= 0 {
		maxEntryBytes = DefaultMaxEntryBytes
	}
	sweepInt := cfg.SweepInterval
	if sweepInt <= 0 {
		sweepInt = sweepInterval
	}
	c := &Cache{
		entries:       make(map[string]*entry),
		closed:        make(chan struct{}),
		ttl:           ttl,
		maxBytes:      maxBytes,
		maxEntryBytes: maxEntryBytes,
		maxEntries:    cfg.MaxEntries,
		sweepInterval: sweepInt,
	}
	go c.sweepLoop()
	return c
}

// DefaultTTL returns the TTL applied to new Set entries.
func (c *Cache) DefaultTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttl
}

// SetDefaultTTL updates the TTL used for subsequent Set calls (existing entries keep their expiry).
func (c *Cache) SetDefaultTTL(ttl time.Duration) {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	c.mu.Lock()
	c.ttl = ttl
	c.mu.Unlock()
}

// MaxEntryBytes returns the per-entry size limit.
func (c *Cache) MaxEntryBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxEntryBytes
}

func (c *Cache) sweepLoop() {
	ticker := time.NewTicker(c.sweepInterval)
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
			c.totalBytes -= int64(len(e.data))
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
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		c.mu.RUnlock()
		c.mu.Lock()
		if e2, ok2 := c.entries[key]; ok2 && time.Now().After(e2.expiresAt) {
			c.totalBytes -= int64(len(e2.data))
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	data := append([]byte(nil), e.data...)
	c.mu.RUnlock()
	return data, true
}

func (c *Cache) Set(_ context.Context, key string, data []byte) error {
	if key == "" {
		return ErrEmptyKey
	}
	entrySize := int64(len(data))
	if entrySize > c.maxEntryBytes {
		return ErrEntryTooLarge
	}

	dataCopy := append([]byte(nil), data...)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxEntries > 0 {
		if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
			return ErrTooManyKeys
		}
	}

	var oldSize int64
	if prev, ok := c.entries[key]; ok {
		oldSize = int64(len(prev.data))
	}
	newTotal := c.totalBytes - oldSize + entrySize
	if newTotal > c.maxBytes {
		return ErrCacheFull
	}

	c.entries[key] = &entry{
		data:      dataCopy,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.totalBytes = newTotal
	return nil
}

func (c *Cache) Invalidate(_ context.Context, prefix string) error {
	if prefix == "" {
		return nil
	}
	c.mu.Lock()
	for k, e := range c.entries {
		if strings.HasPrefix(k, prefix) {
			c.totalBytes -= int64(len(e.data))
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
	return nil
}

// SetLimits updates memory caps at runtime (zero keeps the current value).
func (c *Cache) SetLimits(maxBytes, maxEntryBytes int64, maxEntries int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if maxBytes > 0 {
		c.maxBytes = maxBytes
	}
	if maxEntryBytes > 0 {
		c.maxEntryBytes = maxEntryBytes
	}
	if maxEntries >= 0 {
		c.maxEntries = maxEntries
	}
}
