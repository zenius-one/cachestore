package cachestore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// contract is the behaviour every CacheStore must share.
func contract(t *testing.T, name string, s CacheStore, advance func(time.Duration)) {
	t.Run(name+"/get set delete", func(t *testing.T) {
		if _, ok, err := s.Get(ctx, "k"); ok || err != nil {
			t.Fatalf("miss = %v, %v", ok, err)
		}
		if err := s.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
			t.Fatal(err)
		}
		if v, ok, err := s.Get(ctx, "k"); !ok || err != nil || string(v) != "v" {
			t.Fatalf("hit = %q, %v, %v", v, ok, err)
		}
		if err := s.Delete(ctx, "k", "never-set"); err != nil {
			t.Fatal(err)
		}
		if _, ok, _ := s.Get(ctx, "k"); ok {
			t.Fatal("deleted key still present")
		}
	})
	t.Run(name+"/expiry", func(t *testing.T) {
		_ = s.Set(ctx, "e", []byte("v"), time.Second)
		advance(2 * time.Second)
		if _, ok, _ := s.Get(ctx, "e"); ok {
			t.Fatal("expired key still present")
		}
	})
	t.Run(name+"/setnx", func(t *testing.T) {
		if ok, err := s.SetNX(ctx, "l", []byte("a"), time.Minute); !ok || err != nil {
			t.Fatalf("first SetNX = %v, %v", ok, err)
		}
		if ok, _ := s.SetNX(ctx, "l", []byte("b"), time.Minute); ok {
			t.Fatal("second SetNX must lose")
		}
		if v, _, _ := s.Get(ctx, "l"); string(v) != "a" {
			t.Fatalf("lease holder overwritten: %q", v)
		}
		advance(2 * time.Minute)
		if ok, _ := s.SetNX(ctx, "l", []byte("c"), time.Minute); !ok {
			t.Fatal("SetNX after expiry must win")
		}
	})
	t.Run(name+"/invalid ttl", func(t *testing.T) {
		if err := s.Set(ctx, "x", nil, 0); !errors.Is(err, ErrInvalidTTL) {
			t.Fatalf("err = %v", err)
		}
		if _, err := s.SetNX(ctx, "x", nil, -1); !errors.Is(err, ErrInvalidTTL) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMemoryStoreContract(t *testing.T) {
	now := time.Now()
	s := NewMemoryStore(MemoryOptions{Now: func() time.Time { return now }})
	contract(t, "memory", s, func(d time.Duration) { now = now.Add(d) })
}

func TestRedisStoreContract(t *testing.T) {
	mr := miniredis.RunT(t)
	s := NewRedisStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "t:")
	contract(t, "redis", s, func(d time.Duration) { mr.FastForward(d) })

	_ = s.Set(ctx, "scoped", []byte("v"), time.Minute)
	if !mr.Exists("t:scoped") {
		t.Fatal("key was not stored under the prefix")
	}
}

func TestMemoryStoreIsBoundedAndCopies(t *testing.T) {
	s := NewMemoryStore(MemoryOptions{MaxEntries: 3})
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		_ = s.Set(ctx, k, []byte(k), time.Minute)
	}
	if s.Len() > 3 {
		t.Fatalf("len = %d, want <= 3", s.Len())
	}
	in := []byte("abc")
	_ = s.Set(ctx, "k", in, time.Minute)
	in[0] = 'X'
	got, _, _ := s.Get(ctx, "k")
	got[1] = 'Y'
	again, _, _ := s.Get(ctx, "k")
	if string(again) != "abc" {
		t.Fatalf("store aliased caller memory: %q", again)
	}
}

func TestFallbackDegradesAndRecovers(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	now := time.Now()
	clock := func() time.Time { return now }
	var degraded, recovered int
	f := NewFallback(
		NewRedisStore(redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, DialTimeout: 100 * time.Millisecond}), "f:"),
		NewMemoryStore(MemoryOptions{Now: clock}),
		FallbackOptions{Retry: 5 * time.Second, Now: clock, OnDegraded: func(error) { degraded++ }, OnRecovered: func() { recovered++ }},
	)

	if err := f.Set(ctx, "k", []byte("redis"), time.Minute); err != nil || f.Degraded() {
		t.Fatalf("healthy set: %v degraded=%v", err, f.Degraded())
	}

	mr.Close()
	// Redis is down: reads and writes keep working, from memory.
	if err := f.Set(ctx, "k", []byte("mem"), time.Minute); err != nil {
		t.Fatalf("set while down: %v", err)
	}
	if v, ok, err := f.Get(ctx, "k"); err != nil || !ok || string(v) != "mem" {
		t.Fatalf("get while down = %q, %v, %v", v, ok, err)
	}
	if !f.Degraded() || degraded != 1 {
		t.Fatalf("degraded=%v hooks=%d", f.Degraded(), degraded)
	}
	// Within the retry window the primary is not tried again (no extra transitions or dials).
	_, _, _ = f.Get(ctx, "k")
	if degraded != 1 {
		t.Fatalf("degraded hook fired again: %d", degraded)
	}

	// Redis comes back (new instance, same address); after the window the primary is used again and
	// the memory entry is no longer consulted.
	mr2 := miniredis.NewMiniRedis()
	if err := mr2.StartAddr(addr); err != nil {
		t.Skipf("cannot rebind redis address: %v", err)
	}
	defer mr2.Close()
	now = now.Add(6 * time.Second)
	if _, ok, err := f.Get(ctx, "k"); err != nil || ok {
		t.Fatalf("after recovery the memory entry must not be served: ok=%v err=%v", ok, err)
	}
	if f.Degraded() || recovered != 1 {
		t.Fatalf("degraded=%v recovered=%d", f.Degraded(), recovered)
	}
}

func TestFallbackDeleteClearsBothSides(t *testing.T) {
	mem := NewMemoryStore(MemoryOptions{})
	f := NewFallback(NewMemoryStore(MemoryOptions{}), mem, FallbackOptions{})
	_ = mem.Set(ctx, "k", []byte("stale"), time.Minute)
	_ = f.Delete(ctx, "k")
	if _, ok, _ := mem.Get(ctx, "k"); ok {
		t.Fatal("secondary entry survived a delete")
	}
}

func TestNewWithoutRedisIsMemory(t *testing.T) {
	if _, ok := New(nil, "p:", MemoryOptions{}).(*MemoryStore); !ok {
		t.Fatal("no Redis client must yield a plain MemoryStore")
	}
	mr := miniredis.RunT(t)
	if _, ok := New(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "p:", MemoryOptions{}).(*Fallback); !ok {
		t.Fatal("a Redis client must yield a Fallback")
	}
}

func TestJSONHelpers(t *testing.T) {
	s := NewMemoryStore(MemoryOptions{})
	type v struct{ A int }
	if err := SetJSON(ctx, s, "j", v{A: 7}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := GetJSON[v](ctx, s, "j"); err != nil || !ok || got.A != 7 {
		t.Fatalf("got %+v %v %v", got, ok, err)
	}
	_ = s.Set(ctx, "bad", []byte("not json"), time.Minute)
	if _, ok, err := GetJSON[v](ctx, s, "bad"); ok || err != nil {
		t.Fatalf("undecodable value must be a miss: ok=%v err=%v", ok, err)
	}
}
