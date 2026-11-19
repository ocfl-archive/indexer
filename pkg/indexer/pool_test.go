package indexer

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

func TestKeyedPool(t *testing.T) {
	var callCount int32
	pool := NewKeyedPool(func(key string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		if key == "error" {
			return "", errors.New("cannot create")
		}
		return fmt.Sprintf("val-%s", key), nil
	})

	v1, err := pool.Get("item1")
	if err != nil || v1 != "val-item1" {
		t.Fatalf("unexpected v1: %v, err: %v", v1, err)
	}
	if pool.Len() != 1 {
		t.Fatalf("expected length 1, got %d", pool.Len())
	}

	// Hit cache
	v1cached, err := pool.Get("item1")
	if err != nil || v1cached != "val-item1" {
		t.Fatalf("unexpected cached v1: %v, err: %v", v1cached, err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected factory to be called only once, got %d", callCount)
	}

	// Error path
	_, err = pool.Get("error")
	if err == nil {
		t.Fatalf("expected error from factory")
	}

	// Remove
	removed, ok := pool.Remove("item1")
	if !ok || removed != "val-item1" {
		t.Fatalf("expected remove item1 to succeed, ok=%v, val=%v", ok, removed)
	}
	if pool.Len() != 0 {
		t.Fatalf("expected empty pool after remove")
	}
}

func TestCachePool(t *testing.T) {
	var callCount int32
	pool, err := NewCachePool(50, func(key string) (int, error) {
		atomic.AddInt32(&callCount, 1)
		return len(key), nil
	})
	if err != nil {
		t.Fatalf("failed to create cache pool: %v", err)
	}
	defer pool.Close()

	v1, err := pool.Get("hello")
	if err != nil || v1 != 5 {
		t.Fatalf("expected length 5, got %d, err: %v", v1, err)
	}

	// Cache hit
	v1cached, err := pool.Get("hello")
	if err != nil || v1cached != 5 {
		t.Fatalf("expected length 5 from cache, got %d", v1cached)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected factory call count 1, got %d", callCount)
	}

	pool.Set("manual", 100)
	vManual, err := pool.Get("manual")
	if err != nil || vManual != 100 {
		t.Fatalf("expected manual value 100, got %d", vManual)
	}

	pool.Delete("manual")
}
