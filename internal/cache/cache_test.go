package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGetSetMiss(t *testing.T) {
	c := New()
	t.Cleanup(c.Close)
	ctx := context.Background()

	data, ok := c.Get(ctx, "foo")
	if ok || data != nil {
		t.Fatal("expected miss")
	}

	if err := c.Set(ctx, "foo", []byte("bar")); err != nil {
		t.Fatal(err)
	}
	data, ok = c.Get(ctx, "foo")
	if !ok || string(data) != "bar" {
		t.Fatalf("hit: ok=%v data=%q", ok, data)
	}
}

func TestSetEmptyKey(t *testing.T) {
	c := New()
	t.Cleanup(c.Close)
	if err := c.Set(context.Background(), "", []byte("x")); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestTTLExpiry(t *testing.T) {
	c := NewWithTTL(20 * time.Millisecond)
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("expected expired miss")
	}
}

func TestInvalidatePrefix(t *testing.T) {
	c := New()
	t.Cleanup(c.Close)
	ctx := context.Background()

	_ = c.Set(ctx, "user:1", []byte("a"))
	_ = c.Set(ctx, "user:2", []byte("b"))
	_ = c.Set(ctx, "config:db", []byte("c"))

	if err := c.Invalidate(ctx, "user:"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get(ctx, "user:1"); ok {
		t.Fatal("expected user:1 invalidated")
	}
	if _, ok := c.Get(ctx, "user:2"); ok {
		t.Fatal("expected user:2 invalidated")
	}
	val, ok := c.Get(ctx, "config:db")
	if !ok || string(val) != "c" {
		t.Fatalf("config:db should remain: ok=%v val=%q", ok, val)
	}
}

func TestInvalidateEmptyPrefix(t *testing.T) {
	c := New()
	t.Cleanup(c.Close)
	ctx := context.Background()
	_ = c.Set(ctx, "a", []byte("1"))
	if err := c.Invalidate(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get(ctx, "a"); !ok {
		t.Fatal("expected 'a' to remain")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New()
	t.Cleanup(c.Close)
	ctx := context.Background()
	const goroutines = 10
	const ops = 20

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			prefix := fmt.Sprintf("g%d:", id)
			for j := 0; j < ops; j++ {
				key := prefix + fmt.Sprint(j)
				if err := c.Set(ctx, key, []byte("val")); err != nil {
					t.Errorf("Set: %v", err)
					return
				}
				if _, ok := c.Get(ctx, key); !ok {
					t.Errorf("Get miss for %s", key)
					return
				}
				if j%5 == 0 {
					_ = c.Invalidate(ctx, prefix)
				}
			}
		}(i)
	}
	wg.Wait()
}
