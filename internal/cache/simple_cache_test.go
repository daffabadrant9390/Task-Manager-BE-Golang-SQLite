package cache

import (
	"sync"
	"testing"
	"time"
)

func TestSimpleCache_SetGet_NoTTL(t *testing.T) {
    c := NewSimpleCache[string, int](Options{ConcurrencySafe: false})
    c.Set("a", 1, 0)
    if v, ok := c.Get("a"); !ok || v != 1 {
        t.Fatalf("expected hit with value 1, got ok=%v v=%v", ok, v)
    }
    if !c.Has("a") {
        t.Fatalf("expected Has to be true")
    }
    if c.Len() != 1 {
        t.Fatalf("expected Len=1, got %d", c.Len())
    }
}

func TestSimpleCache_TTL_Expiry(t *testing.T) {
    c := NewSimpleCache[string, string](Options{ConcurrencySafe: true})

    // Freeze time via now indirection
    base := time.Now()
    now = func() time.Time { return base }
    t.Cleanup(func() { now = time.Now })

    c.Set("k", "v", time.Second)
    if v, ok := c.Get("k"); !ok || v != "v" {
        t.Fatalf("expected hit before expiry")
    }

    // advance time beyond TTL
    base = base.Add(2 * time.Second)
    if _, ok := c.Get("k"); ok {
        t.Fatalf("expected miss after expiry")
    }
    if c.Has("k") {
        t.Fatalf("expected Has=false after expiry")
    }
    c.PurgeExpired()
    if c.Len() != 0 {
        t.Fatalf("expected Len=0 after purge, got %d", c.Len())
    }
}

func TestSimpleCache_Delete_Clear(t *testing.T) {
    c := NewSimpleCache[int, int](Options{ConcurrencySafe: true})
    c.Set(1, 10, 0)
    c.Set(2, 20, 0)
    c.Delete(1)
    if _, ok := c.Get(1); ok {
        t.Fatalf("expected key 1 to be deleted")
    }
    if c.Len() != 1 {
        t.Fatalf("expected Len=1, got %d", c.Len())
    }
    c.Clear()
    if c.Len() != 0 {
        t.Fatalf("expected Len=0 after Clear, got %d", c.Len())
    }
}

func TestSimpleCache_ConcurrencySafe_Toggle(t *testing.T) {
    // This test stresses safe mode with concurrency, and unsafe mode sequentially.
    keys := 100
    rounds := 200

    // Safe: allow concurrent writers/readers.
    {
        c := NewSimpleCache[int, int](Options{ConcurrencySafe: true})
        var wg sync.WaitGroup
        for i := 0; i < keys; i++ {
            i := i
            wg.Add(1)
            go func() {
                defer wg.Done()
                for r := 0; r < rounds; r++ {
                    c.Set(i, r, 0)
                    _, _ = c.Get(i)
                }
            }()
        }
        wg.Wait()
        for i := 0; i < keys; i++ {
            if _, ok := c.Get(i); !ok {
                t.Fatalf("expected ok in safe mode")
            }
        }
    }

    // Unsafe: exercise API sequentially to confirm it works (no data races expected).
    {
        c := NewSimpleCache[int, int](Options{ConcurrencySafe: false})
        for i := 0; i < keys; i++ {
            for r := 0; r < rounds; r++ {
                c.Set(i, r, 0)
                _, _ = c.Get(i)
            }
        }
        for i := 0; i < keys; i++ {
            if _, ok := c.Get(i); !ok {
                t.Fatalf("expected ok in unsafe mode (sequential)")
            }
        }
    }
}


