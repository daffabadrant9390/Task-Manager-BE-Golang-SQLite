package cache

import "time"

// Cache defines a minimal key-value cache API with optional TTL per entry.
// Implementations may or may not be goroutine-safe depending on configuration.
type Cache[K comparable, V any] interface {
    // Get returns the value and whether it was present and not expired.
    Get(key K) (V, bool)

    // Set stores the value with an optional TTL. If ttl <= 0, the entry does not expire.
    Set(key K, value V, ttl time.Duration)

    // Delete removes a key if present.
    Delete(key K)

    // Has reports whether a key is present and not expired.
    Has(key K) bool

    // Len returns the number of non-expired items currently stored.
    Len() int

    // Clear removes all entries.
    Clear()

    // PurgeExpired scans and removes expired entries.
    PurgeExpired()
}


