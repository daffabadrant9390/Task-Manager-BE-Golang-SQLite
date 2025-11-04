package cache

import (
	"sync"
	"time"
)

// entry stores a cached value and its absolute expiration timestamp.
type entry[V any] struct {
    value      V
    expiresAt  time.Time // zero means no expiration
}

// SimpleCache is a lightweight map-backed cache with optional concurrency safety.
// It supports per-item TTL (no background janitor; cleanup is lazy or via PurgeExpired).
type SimpleCache[K comparable, V any] struct {
    // If muPtr is nil, the cache is NOT goroutine-safe.
    // If muPtr is non-nil, it guards all operations.
    muPtr *sync.RWMutex

    items map[K]entry[V]
}

// Options controls construction of a SimpleCache.
type Options struct {
    // ConcurrencySafe controls whether operations are guarded by a RWMutex.
    // If false, the cache is not safe for concurrent use and may be faster in single-threaded contexts.
    ConcurrencySafe bool
}

// NewSimpleCache constructs a new SimpleCache with the given options.
func NewSimpleCache[K comparable, V any](opts Options) *SimpleCache[K, V] {
    var mu *sync.RWMutex
    if opts.ConcurrencySafe {
        mu = &sync.RWMutex{}
    }
    return &SimpleCache[K, V]{
        muPtr: mu,
        items: make(map[K]entry[V]),
    }
}

func (c *SimpleCache[K, V]) lockR() func() {
    if c.muPtr == nil {
        return func() {}
    }
    c.muPtr.RLock()
    return c.muPtr.RUnlock
}

func (c *SimpleCache[K, V]) lockW() func() {
    if c.muPtr == nil {
        return func() {}
    }
    c.muPtr.Lock()
    return c.muPtr.Unlock
}

// now is a small indirection to allow test stubbing if needed.
var now = time.Now

// Get implements Cache.Get.
func (c *SimpleCache[K, V]) Get(key K) (V, bool) {
    unlock := c.lockR()
    defer unlock()

    var zero V
    e, ok := c.items[key]
    if !ok {
        return zero, false
    }
    if !e.expiresAt.IsZero() && now().After(e.expiresAt) {
        // expired; treat as miss (lazy cleanup deferred to PurgeExpired)
        return zero, false
    }
    return e.value, true
}

// Set implements Cache.Set.
func (c *SimpleCache[K, V]) Set(key K, value V, ttl time.Duration) {
    unlock := c.lockW()
    defer unlock()

    var exp time.Time
    if ttl > 0 {
        exp = now().Add(ttl)
    }
    c.items[key] = entry[V]{
        value:     value,
        expiresAt: exp,
    }
}

// Delete implements Cache.Delete.
func (c *SimpleCache[K, V]) Delete(key K) {
    unlock := c.lockW()
    defer unlock()
    delete(c.items, key)
}

// Has implements Cache.Has.
func (c *SimpleCache[K, V]) Has(key K) bool {
    unlock := c.lockR()
    defer unlock()
    e, ok := c.items[key]
    if !ok {
        return false
    }
    if !e.expiresAt.IsZero() && now().After(e.expiresAt) {
        return false
    }
    return true
}

// Len implements Cache.Len. It counts only non-expired entries.
func (c *SimpleCache[K, V]) Len() int {
    unlock := c.lockR()
    defer unlock()
    count := 0
    for _, e := range c.items {
        if e.expiresAt.IsZero() || now().Before(e.expiresAt) {
            count++
        }
    }
    return count
}

// Clear implements Cache.Clear.
func (c *SimpleCache[K, V]) Clear() {
    unlock := c.lockW()
    defer unlock()
    c.items = make(map[K]entry[V])
}

// PurgeExpired implements Cache.PurgeExpired.
func (c *SimpleCache[K, V]) PurgeExpired() {
    unlock := c.lockW()
    defer unlock()
    if len(c.items) == 0 {
        return
    }
    nowTs := now()
    for k, e := range c.items {
        if !e.expiresAt.IsZero() && nowTs.After(e.expiresAt) {
            delete(c.items, k)
        }
    }
}

// Ensure SimpleCache implements Cache at compile time.
var _ Cache[any, any] = (*SimpleCache[any, any])(nil)


