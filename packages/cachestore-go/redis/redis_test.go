package cacheredis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	cachestore "github.com/zenius-one/cachestore/packages/cachestore-go"
)

var ctx = context.Background()

func TestContract(t *testing.T) {
	mr := miniredis.RunT(t)
	s := NewRedisStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "t:")

	if _, ok, err := s.Get(ctx, "k"); ok || err != nil {
		t.Fatalf("miss = %v, %v", ok, err)
	}
	if err := s.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if !mr.Exists("t:k") {
		t.Fatal("key was not stored under the prefix")
	}
	if v, ok, _ := s.Get(ctx, "k"); !ok || string(v) != "v" {
		t.Fatalf("hit = %q, %v", v, ok)
	}
	_ = s.Delete(ctx, "k", "never-set")
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("deleted key still present")
	}

	_ = s.Set(ctx, "e", []byte("v"), time.Second)
	mr.FastForward(2 * time.Second)
	if _, ok, _ := s.Get(ctx, "e"); ok {
		t.Fatal("expired key still present")
	}

	if ok, _ := s.SetNX(ctx, "l", []byte("a"), time.Minute); !ok {
		t.Fatal("first SetNX must win")
	}
	if ok, _ := s.SetNX(ctx, "l", []byte("b"), time.Minute); ok {
		t.Fatal("second SetNX must lose")
	}
	if v, _, _ := s.Get(ctx, "l"); string(v) != "a" {
		t.Fatalf("lease holder overwritten: %q", v)
	}
	mr.FastForward(2 * time.Minute)
	if ok, _ := s.SetNX(ctx, "l", []byte("c"), time.Minute); !ok {
		t.Fatal("SetNX after expiry must win")
	}

	if err := s.Set(ctx, "x", nil, 0); err != cachestore.ErrInvalidTTL {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.SetNX(ctx, "x", nil, -1); err != cachestore.ErrInvalidTTL {
		t.Fatalf("err = %v", err)
	}
}

func TestNew(t *testing.T) {
	if _, ok := New(nil, "p:", cachestore.MemoryOptions{}).(*cachestore.MemoryStore); !ok {
		t.Fatal("no Redis client must yield a plain MemoryStore")
	}
	mr := miniredis.RunT(t)
	if _, ok := New(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "p:", cachestore.MemoryOptions{}).(*cachestore.Fallback); !ok {
		t.Fatal("a Redis client must yield a Fallback")
	}
}

// With Redis gone, the composed store keeps working from memory and recovers when Redis returns.
func TestFallbackSurvivesRedisOutageAndRecovers(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	now := time.Now()
	client := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	var recovered int
	f := New(client, "f:", cachestore.MemoryOptions{Now: func() time.Time { return now }},
		cachestore.FallbackOptions{Retry: 5 * time.Second, Now: func() time.Time { return now }, OnRecovered: func() { recovered++ }}).(*cachestore.Fallback)

	_ = f.Set(ctx, "k", []byte("redis"), time.Minute)
	mr.Close()
	if err := f.Set(ctx, "k", []byte("mem"), time.Minute); err != nil {
		t.Fatalf("set while Redis is down: %v", err)
	}
	if v, ok, err := f.Get(ctx, "k"); err != nil || !ok || string(v) != "mem" {
		t.Fatalf("get while down = %q, %v, %v", v, ok, err)
	}
	if !f.Degraded() {
		t.Fatal("expected degraded")
	}

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
