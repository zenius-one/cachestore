package cachestore

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestFallbackDeleteClearsBothSides(t *testing.T) {
	mem := NewMemoryStore(MemoryOptions{})
	f := NewFallback(NewMemoryStore(MemoryOptions{}), mem, FallbackOptions{})
	_ = mem.Set(ctx, "k", []byte("stale"), time.Minute)
	_ = f.Delete(ctx, "k")
	if _, ok, _ := mem.Get(ctx, "k"); ok {
		t.Fatal("secondary entry survived a delete")
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

// flaky is a CacheStore whose availability a test controls.
type flaky struct {
	*MemoryStore
	down bool
	hits int
}

func (f *flaky) check() error {
	f.hits++
	if f.down {
		return errors.New("primary down")
	}
	return nil
}

func (f *flaky) Get(c context.Context, k string) ([]byte, bool, error) {
	if err := f.check(); err != nil {
		return nil, false, err
	}
	return f.MemoryStore.Get(c, k)
}

func (f *flaky) Set(c context.Context, k string, v []byte, ttl time.Duration) error {
	if err := f.check(); err != nil {
		return err
	}
	return f.MemoryStore.Set(c, k, v, ttl)
}

func (f *flaky) SetNX(c context.Context, k string, v []byte, ttl time.Duration) (bool, error) {
	if err := f.check(); err != nil {
		return false, err
	}
	return f.MemoryStore.SetNX(c, k, v, ttl)
}

func (f *flaky) Delete(c context.Context, keys ...string) error {
	if err := f.check(); err != nil {
		return err
	}
	return f.MemoryStore.Delete(c, keys...)
}

func TestFallbackDegradesAndRecovers(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	primary := &flaky{MemoryStore: NewMemoryStore(MemoryOptions{Now: clock})}
	var degraded, recovered int
	f := NewFallback(primary, NewMemoryStore(MemoryOptions{Now: clock}),
		FallbackOptions{Retry: 5 * time.Second, Now: clock, OnDegraded: func(error) { degraded++ }, OnRecovered: func() { recovered++ }})

	if err := f.Set(ctx, "k", []byte("primary"), time.Minute); err != nil || f.Degraded() {
		t.Fatalf("healthy set: %v degraded=%v", err, f.Degraded())
	}

	primary.down = true
	// The primary is down: reads and writes keep working, from the secondary.
	if err := f.Set(ctx, "k", []byte("secondary"), time.Minute); err != nil {
		t.Fatalf("set while down: %v", err)
	}
	if v, ok, err := f.Get(ctx, "k"); err != nil || !ok || string(v) != "secondary" {
		t.Fatalf("get while down = %q, %v, %v", v, ok, err)
	}
	if !f.Degraded() || degraded != 1 {
		t.Fatalf("degraded=%v hooks=%d", f.Degraded(), degraded)
	}
	// Inside the retry window the primary is not tried again.
	before := primary.hits
	_, _, _ = f.Get(ctx, "k")
	if primary.hits != before || degraded != 1 {
		t.Fatalf("primary tried inside the window: hits %d->%d degraded hook %d", before, primary.hits, degraded)
	}

	// After the window, with the primary back, it is the source of truth again and the secondary
	// entry is no longer served.
	primary.down = false
	now = now.Add(6 * time.Second)
	if v, ok, err := f.Get(ctx, "k"); err != nil || !ok || string(v) != "primary" {
		t.Fatalf("after recovery got %q, %v, %v; want the primary's value", v, ok, err)
	}
	if f.Degraded() || recovered != 1 {
		t.Fatalf("degraded=%v recovered=%d", f.Degraded(), recovered)
	}
}
