// Copyright 2026 Rahmad Afandi. MIT License.

// Package cache is a generic in-process cache with TTL, LRU max-size eviction,
// and singleflight load deduplication. It complements redis.Remember for hot
// data that should not cost a network round-trip per read.
package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// config holds the non-generic options for a Cache.
type config struct {
	maxSize    int
	defaultTTL time.Duration
	janitor    time.Duration
}

// Option configures a Cache.
type Option func(*config)

// WithMaxSize bounds the number of entries; the least-recently-used is evicted
// when the limit is exceeded. Zero (default) is unbounded.
func WithMaxSize(n int) Option { return func(c *config) { c.maxSize = n } }

// WithDefaultTTL sets the TTL applied by Set and GetOrLoad. Zero (default) means
// entries do not expire.
func WithDefaultTTL(d time.Duration) Option { return func(c *config) { c.defaultTTL = d } }

// WithJanitor runs a background goroutine that sweeps expired entries every
// interval. Without it, expiry is lazy (checked on Get). Call Close to stop it.
func WithJanitor(interval time.Duration) Option { return func(c *config) { c.janitor = interval } }

type entry[V any] struct {
	key       string
	value     V
	expiresAt time.Time // zero = never expires
	el        *list.Element
}

// Cache is a generic in-process cache.
type Cache[V any] struct {
	mu         sync.Mutex
	items      map[string]*entry[V]
	lru        *list.List // front = most recently used; values are *entry[V]
	maxSize    int
	defaultTTL time.Duration
	group      singleflight.Group
	now        func() time.Time

	janitorStop chan struct{}
	closeOnce   sync.Once
}

// New returns a cache for values of type V.
func New[V any](opts ...Option) *Cache[V] {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	c := &Cache[V]{
		items:      make(map[string]*entry[V]),
		lru:        list.New(),
		maxSize:    cfg.maxSize,
		defaultTTL: cfg.defaultTTL,
		now:        time.Now,
	}
	if cfg.janitor > 0 {
		c.janitorStop = make(chan struct{})
		go c.runJanitor(cfg.janitor, c.janitorStop)
	}
	return c
}

// Get returns the value for key and whether it was present and unexpired.
func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	if c.expired(e) {
		c.removeLocked(e)
		var zero V
		return zero, false
	}
	c.lru.MoveToFront(e.el)
	return e.value, true
}

// Set stores val under key using the default TTL.
func (c *Cache[V]) Set(key string, val V) {
	c.SetTTL(key, val, c.defaultTTL)
}

// SetTTL stores val under key with an explicit TTL; ttl <= 0 means no expiry.
func (c *Cache[V]) SetTTL(key string, val V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(key, val, ttl)
}

func (c *Cache[V]) setLocked(key string, val V, ttl time.Duration) {
	var exp time.Time
	if ttl > 0 {
		exp = c.now().Add(ttl)
	}
	if e, ok := c.items[key]; ok {
		e.value = val
		e.expiresAt = exp
		c.lru.MoveToFront(e.el)
		return
	}
	e := &entry[V]{key: key, value: val, expiresAt: exp}
	e.el = c.lru.PushFront(e)
	c.items[key] = e
	if c.maxSize > 0 && c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
}

// Delete removes key.
func (c *Cache[V]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		c.removeLocked(e)
	}
}

// Len returns the number of entries (including any not yet swept that may be
// expired).
func (c *Cache[V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

// GetOrLoad returns the cached value for key, or runs loader once (deduplicating
// concurrent callers), caches the result under the default TTL, and returns it.
// A loader error is returned and nothing is cached.
func (c *Cache[V]) GetOrLoad(ctx context.Context, key string, loader func() (V, error)) (V, error) {
	if v, ok := c.Get(key); ok {
		return v, nil
	}
	v, err, _ := c.group.Do(key, func() (any, error) {
		if v, ok := c.Get(key); ok {
			return v, nil
		}
		val, err := loader()
		if err != nil {
			return nil, err
		}
		c.Set(key, val)
		return val, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return v.(V), nil
}

// Close stops the janitor goroutine, if any. It is safe to call multiple times.
func (c *Cache[V]) Close() {
	if c.janitorStop != nil {
		c.closeOnce.Do(func() { close(c.janitorStop) })
	}
}

func (c *Cache[V]) expired(e *entry[V]) bool {
	return !e.expiresAt.IsZero() && !c.now().Before(e.expiresAt)
}

func (c *Cache[V]) removeLocked(e *entry[V]) {
	c.lru.Remove(e.el)
	delete(c.items, e.key)
}

func (c *Cache[V]) evictOldest() {
	back := c.lru.Back()
	if back == nil {
		return
	}
	c.removeLocked(back.Value.(*entry[V]))
}

func (c *Cache[V]) runJanitor(interval time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			c.sweep()
		}
	}
}

func (c *Cache[V]) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.items {
		if c.expired(e) {
			c.removeLocked(e)
		}
	}
}
