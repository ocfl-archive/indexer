package indexer

import (
	"context"
	"sync"

	"github.com/maypok86/otter/v2"
)

// KeyedPool manages shared, lazily-instantiated resources identified by key K using an in-memory map.
type KeyedPool[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]V
	factory func(key K) (V, error)
}

// NewKeyedPool creates a new KeyedPool with a factory function for resource creation.
func NewKeyedPool[K comparable, V any](factory func(key K) (V, error)) *KeyedPool[K, V] {
	return &KeyedPool[K, V]{
		entries: make(map[K]V),
		factory: factory,
	}
}

// Get returns the resource associated with key, creating it via factory if not present.
func (p *KeyedPool[K, V]) Get(key K) (V, error) {
	p.mu.RLock()
	if val, ok := p.entries[key]; ok {
		p.mu.RUnlock()
		return val, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check after acquiring write lock
	if val, ok := p.entries[key]; ok {
		return val, nil
	}

	val, err := p.factory(key)
	if err != nil {
		var zero V
		return zero, err
	}
	p.entries[key] = val
	return val, nil
}

// Remove removes and returns a resource from the pool, if present.
func (p *KeyedPool[K, V]) Remove(key K) (V, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	val, ok := p.entries[key]
	if ok {
		delete(p.entries, key)
	}
	return val, ok
}

// Clear removes all resources from the pool.
func (p *KeyedPool[K, V]) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = make(map[K]V)
}

// Len returns the current number of cached items in the pool.
func (p *KeyedPool[K, V]) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.entries)
}

// CachePool is a high-performance generic resource and connection pool backed by Otter's S3-FIFO cache.
// It provides lock-free concurrent reads, automatic eviction, and optional capacity bounding.
type CachePool[K comparable, V any] struct {
	cache   *otter.Cache[K, V]
	factory func(key K) (V, error)
	loader  otter.LoaderFunc[K, V]
}

// NewCachePool creates a new high-performance CachePool backed by Otter with the specified capacity limit.
func NewCachePool[K comparable, V any](capacity int, factory func(key K) (V, error)) (*CachePool[K, V], error) {
	if capacity <= 0 {
		capacity = 1000
	}
	cache, err := otter.New(&otter.Options[K, V]{
		MaximumSize: capacity,
	})
	if err != nil {
		return nil, err
	}
	loader := otter.LoaderFunc[K, V](func(ctx context.Context, key K) (V, error) {
		return factory(key)
	})
	return &CachePool[K, V]{
		cache:   cache,
		factory: factory,
		loader:  loader,
	}, nil
}

// Get retrieves an item from the cache lock-free, or loads it via factory on a miss.
func (cp *CachePool[K, V]) Get(key K) (V, error) {
	return cp.GetContext(context.Background(), key)
}

// GetContext retrieves an item from the cache with context support.
func (cp *CachePool[K, V]) GetContext(ctx context.Context, key K) (V, error) {
	return cp.cache.Get(ctx, key, cp.loader)
}

// Set stores a value directly into the cache.
func (cp *CachePool[K, V]) Set(key K, val V) (V, bool) {
	return cp.cache.Set(key, val)
}

// Delete removes a key from the cache.
func (cp *CachePool[K, V]) Delete(key K) (V, bool) {
	return cp.cache.Invalidate(key)
}

// Clear invalidates all entries in the cache.
func (cp *CachePool[K, V]) Clear() {
	cp.cache.InvalidateAll()
}

// Close stops background goroutines of the cache.
func (cp *CachePool[K, V]) Close() bool {
	return cp.cache.StopAllGoroutines()
}

// Len returns the estimated number of items currently in the cache.
func (cp *CachePool[K, V]) Len() int {
	return cp.cache.EstimatedSize()
}
