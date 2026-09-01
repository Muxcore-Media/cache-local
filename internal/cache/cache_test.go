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

func TestSweeperRemovesExpiredWithoutGet(t *testing.T) {
	c := NewWithConfig(Config{TTL: 15 * time.Millisecond, SweepInterval: 10 * time.Millisecond})
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "sweep-me", []byte("v")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(70 * time.Millisecond)

	c.mu.RLock()
	_, exists := c.entries["sweep-me"]
	c.mu.RUnlock()
	if exists {
		t.Fatal("expected sweeper to remove expired key without Get")
	}
}

func TestSetDefaultTTLDoesNotRewriteExistingExpiry(t *testing.T) {
	c := NewWithTTL(200 * time.Millisecond)
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	c.SetDefaultTTL(5 * time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	if _, ok := c.Get(ctx, "k"); !ok {
		t.Fatal("expected original TTL to remain after SetDefaultTTL")
	}
}

func TestExpireDeleteDoesNotClobberConcurrentSet(t *testing.T) {
	c := NewWithTTL(30 * time.Millisecond)
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "race", []byte("old")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(35 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = c.Get(ctx, "race")
	}()
	go func() {
		defer wg.Done()
		_ = c.Set(ctx, "race", []byte("new"))
	}()
	wg.Wait()

	got, ok := c.Get(ctx, "race")
	if !ok || string(got) != "new" {
		t.Fatalf("concurrent Set clobbered: ok=%v got=%q", ok, got)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	c := New()
	t.Cleanup(c.Close)
	ctx := context.Background()

	orig := []byte("data")
	if err := c.Set(ctx, "k", orig); err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get(ctx, "k")
	if !ok {
		t.Fatal("expected hit")
	}
	got[0] = 'X'
	got2, ok := c.Get(ctx, "k")
	if !ok || string(got2) != "data" {
		t.Fatalf("Get returned shared buffer: %q", got2)
	}
}

func TestMaxEntryBytes(t *testing.T) {
	c := NewWithConfig(Config{TTL: time.Minute, MaxEntryBytes: 4, MaxBytes: 100})
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("12345")); err == nil {
		t.Fatal("expected entry too large error")
	}
	if err := c.Set(ctx, "k", []byte("1234")); err != nil {
		t.Fatal(err)
	}
}

func TestMaxBytes(t *testing.T) {
	c := NewWithConfig(Config{TTL: time.Minute, MaxEntryBytes: 10, MaxBytes: 12})
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "a", []byte("123456")); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "b", []byte("1234567")); err == nil {
		t.Fatal("expected cache full error")
	}
}

func TestMaxEntries(t *testing.T) {
	c := NewWithConfig(Config{TTL: time.Minute, MaxEntries: 2, MaxBytes: 1 << 20, MaxEntryBytes: 64})
	t.Cleanup(c.Close)
	ctx := context.Background()

	if err := c.Set(ctx, "a", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "b", []byte("2")); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "c", []byte("3")); err == nil {
		t.Fatal("expected too many keys error")
	}
	if err := c.Set(ctx, "a", []byte("updated")); err != nil {
		t.Fatal("expected update of existing key to succeed")
	}
}
