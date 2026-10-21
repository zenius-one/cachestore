package resilient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	cachestore "github.com/zenius-one/cachestore/packages/cachestore-go"
)

type recordedLog struct {
	mu          sync.Mutex
	warns, info []string
}

func (r *recordedLog) logger() Logger {
	return Logger{
		Warnf: func(f string, a ...any) { r.mu.Lock(); r.warns = append(r.warns, fmt.Sprintf(f, a...)); r.mu.Unlock() },
		Infof: func(f string, a ...any) { r.mu.Lock(); r.info = append(r.info, fmt.Sprintf(f, a...)); r.mu.Unlock() },
	}
}

func (r *recordedLog) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.warns), len(r.info)
}

// flaky stands in for Redis: fails while down is set.
type flaky struct {
	cachestore.CacheStore
	mu   sync.Mutex
	down bool
}

func newFlaky(down bool) *flaky {
	return &flaky{CacheStore: cachestore.NewMemoryStore(cachestore.MemoryOptions{}), down: down}
}
func (f *flaky) setDown(v bool) { f.mu.Lock(); f.down = v; f.mu.Unlock() }
func (f *flaky) err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.down {
		return errors.New("connection refused")
	}
	return nil
}
func (f *flaky) Get(ctx context.Context, k string) ([]byte, bool, error) {
	if err := f.err(); err != nil {
		return nil, false, err
	}
	return f.CacheStore.Get(ctx, k)
}
func (f *flaky) Set(ctx context.Context, k string, v []byte, ttl time.Duration) error {
	if err := f.err(); err != nil {
		return err
	}
	return f.CacheStore.Set(ctx, k, v, ttl)
}
func (f *flaky) SetNX(ctx context.Context, k string, v []byte, ttl time.Duration) (bool, error) {
	if err := f.err(); err != nil {
		return false, err
	}
	return f.CacheStore.SetNX(ctx, k, v, ttl)
}
func (f *flaky) Delete(ctx context.Context, k ...string) error {
	if err := f.err(); err != nil {
		return err
	}
	return f.CacheStore.Delete(ctx, k...)
}

// slow blocks until the context ends, like a backend that stopped answering.
type slow struct{ cachestore.CacheStore }

func (slow) Get(ctx context.Context, _ string) ([]byte, bool, error) {
	<-ctx.Done()
	return nil, false, ctx.Err()
}

func TestBounded_should_give_each_operation_its_own_deadline(t *testing.T) {
	s := Bounded(slow{cachestore.NewMemoryStore(cachestore.MemoryOptions{})}, 30*time.Millisecond)
	start := time.Now()
	_, _, err := s.Get(context.Background(), "k")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("a stuck backend must be cut off at the per-operation timeout")
	}
}

func TestBounded_should_not_extend_a_shorter_caller_deadline(t *testing.T) {
	s := Bounded(slow{cachestore.NewMemoryStore(cachestore.MemoryOptions{})}, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, _, err := s.Get(ctx, "k"); !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("err = %v after %s, want the caller's deadline to win", err, time.Since(start))
	}
}

func TestBounded_with_no_timeout_should_return_the_store_unchanged(t *testing.T) {
	m := cachestore.NewMemoryStore(cachestore.MemoryOptions{})
	if Bounded(m, 0) != cachestore.CacheStore(m) {
		t.Fatal("timeout <= 0 must be a no-op")
	}
}

func TestLogged_should_rate_limit_but_count_what_it_suppressed(t *testing.T) {
	var log recordedLog
	now := time.Now()
	l := newLogged(newFlaky(true), "redis", log.logger(), time.Minute, func() time.Time { return now })
	for range 100 {
		_, _, _ = l.Get(context.Background(), "k")
	}
	if w, _ := log.counts(); w != 1 {
		t.Fatalf("warnings = %d for 100 errors inside one interval, want 1", w)
	}
	now = now.Add(2 * time.Minute)
	_, _, _ = l.Get(context.Background(), "k")
	log.mu.Lock()
	defer log.mu.Unlock()
	if len(log.warns) != 2 || !strings.Contains(log.warns[1], "+99 similar") {
		t.Fatalf("warnings = %q, want a second one reporting the 99 suppressed errors", log.warns)
	}
}

func TestLogged_should_not_blame_the_backend_for_a_cancelled_caller(t *testing.T) {
	var log recordedLog
	l := Logged(newFlaky(true), "redis", log.logger(), time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _ = l.Get(ctx, "k")
	if w, _ := log.counts(); w != 0 {
		t.Fatalf("warnings = %d, want 0 for a cancelled caller", w)
	}
}

func TestLogged_should_name_the_failed_operation(t *testing.T) {
	var log recordedLog
	l := Logged(newFlaky(true), "redis", log.logger(), time.Nanosecond)
	ctx := context.Background()
	_ = l.Set(ctx, "k", nil, time.Minute)
	time.Sleep(2 * time.Millisecond)
	_, _ = l.SetNX(ctx, "k", nil, time.Minute)
	time.Sleep(2 * time.Millisecond)
	_ = l.Delete(ctx, "k")
	log.mu.Lock()
	defer log.mu.Unlock()
	got := strings.Join(log.warns, "|")
	for _, want := range []string{"redis set failed", "redis setnx failed", "redis delete failed"} {
		if !strings.Contains(got, want) {
			t.Errorf("warnings %q missing %q", got, want)
		}
	}
}

func TestNew_should_report_an_outage_and_the_recovery(t *testing.T) {
	primary := newFlaky(true)
	var log recordedLog
	var mu sync.Mutex
	var events []string
	store := New(primary, Options{Name: "redis", Log: log.logger(), Retry: 20 * time.Millisecond,
		OnDegraded:  func(error) { mu.Lock(); events = append(events, "degraded"); mu.Unlock() },
		OnRecovered: func() { mu.Lock(); events = append(events, "recovered"); mu.Unlock() }})
	ctx := context.Background()

	if err := store.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("a dead primary must not fail the cache call: %v", err)
	}
	if v, ok, _ := store.Get(ctx, "k"); !ok || string(v) != "v" {
		t.Fatal("while degraded the store should serve from local memory")
	}
	if !store.Degraded() {
		t.Fatal("Degraded() should be true")
	}
	if w, _ := log.counts(); w < 2 {
		t.Fatalf("warnings = %d, want the primary error and the degrade notice", w)
	}

	primary.setDown(false)
	time.Sleep(40 * time.Millisecond) // past the retry window
	if _, _, err := store.Get(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(events) != "[degraded recovered]" {
		t.Fatalf("events = %v, want [degraded recovered]", events)
	}
	if _, i := log.counts(); i != 1 {
		t.Fatalf("recovery notices = %d, want 1", i)
	}
}

func TestProber_should_detect_recovery_without_any_other_traffic(t *testing.T) {
	primary := newFlaky(true)
	fb := New(primary, Options{Retry: 10 * time.Millisecond})
	p := NewProber(fb, 10*time.Millisecond, 0)
	_, _, _ = fb.Get(context.Background(), "k") // discovers the outage
	if !p.Active() {
		t.Fatal("outage should be active")
	}
	primary.setDown(false)
	deadline := time.Now().Add(2 * time.Second)
	for p.Active() {
		if time.Now().After(deadline) {
			t.Fatal("never recovered: Probe must give the Fallback a chance to retry")
		}
		p.Probe()
		time.Sleep(15 * time.Millisecond)
	}
}

func TestProber_should_be_safe_on_nil(t *testing.T) {
	var p *Prober
	p.Probe()
	if p.Active() {
		t.Fatal("a nil Prober is never active")
	}
}
