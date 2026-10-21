package resilient

import (
	"context"
	"errors"
	"sync"
	"time"

	cachestore "github.com/zenius-one/cachestore/packages/cachestore-go"
)

// Bounded returns a CacheStore in which every operation gets its own deadline of timeout, derived
// from the caller's context (whichever expires first wins). A client's own timeouts are usually
// seconds, far too long for a cache sitting in front of a request: a backend that stops answering
// must degrade the cache within a fraction of a second. A timeout is an ordinary error to the
// store beneath, so a Fallback there degrades and a breaker above it opens. timeout <= 0 returns
// store unchanged.
func Bounded(store cachestore.CacheStore, timeout time.Duration) cachestore.CacheStore {
	if timeout <= 0 {
		return store
	}
	return bounded{CacheStore: store, timeout: timeout}
}

type bounded struct {
	cachestore.CacheStore
	timeout time.Duration
}

func (b bounded) Get(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	return b.CacheStore.Get(ctx, key)
}

func (b bounded) Set(ctx context.Context, key string, v []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	return b.CacheStore.Set(ctx, key, v, ttl)
}

func (b bounded) SetNX(ctx context.Context, key string, v []byte, ttl time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	return b.CacheStore.SetNX(ctx, key, v, ttl)
}

func (b bounded) Delete(ctx context.Context, keys ...string) error {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	return b.CacheStore.Delete(ctx, keys...)
}

// Logger is where the wrappers report. Both funcs are optional.
type Logger struct {
	Warnf func(format string, args ...any)
	Infof func(format string, args ...any)
}

func (l Logger) warn(format string, args ...any) {
	if l.Warnf != nil {
		l.Warnf(format, args...)
	}
}

func (l Logger) info(format string, args ...any) {
	if l.Infof != nil {
		l.Infof(format, args...)
	}
}

// DefaultLogEvery is the shortest gap between two "store failed" warnings from Logged.
const DefaultLogEvery = 30 * time.Second

// Logged returns a CacheStore that logs the errors of store, at most one warning per every (0 means
// DefaultLogEvery); errors in between are counted and reported in the next warning, so a dead
// backend cannot flood the log at request rate. cachestore.Fallback swallows primary errors and
// only announces the first failure of an outage, so without this a long outage, or a backend that
// is slow rather than down, leaves a single log line. A context cancelled by the caller is not the
// backend's fault and is not logged. name labels the store in messages ("redis").
func Logged(store cachestore.CacheStore, name string, log Logger, every time.Duration) cachestore.CacheStore {
	return newLogged(store, name, log, every, time.Now)
}

func newLogged(store cachestore.CacheStore, name string, log Logger, every time.Duration, now func() time.Time) *logged {
	if every <= 0 {
		every = DefaultLogEvery
	}
	return &logged{CacheStore: store, name: name, log: log, every: every, now: now}
}

type logged struct {
	cachestore.CacheStore
	name  string
	log   Logger
	every time.Duration
	now   func() time.Time

	mu         sync.Mutex
	last       time.Time
	suppressed int
}

func (l *logged) note(op string, ctx context.Context, err error) {
	if err == nil || errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	l.mu.Lock()
	now := l.now()
	if !l.last.IsZero() && now.Sub(l.last) < l.every {
		l.suppressed++
		l.mu.Unlock()
		return
	}
	n := l.suppressed
	l.last, l.suppressed = now, 0
	l.mu.Unlock()
	if n > 0 {
		l.log.warn("cache: %s %s failed: %v (+%d similar errors in the last %s)", l.name, op, err, n, l.every)
		return
	}
	l.log.warn("cache: %s %s failed: %v", l.name, op, err)
}

func (l *logged) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, ok, err := l.CacheStore.Get(ctx, key)
	l.note("get", ctx, err)
	return v, ok, err
}

func (l *logged) Set(ctx context.Context, key string, v []byte, ttl time.Duration) error {
	err := l.CacheStore.Set(ctx, key, v, ttl)
	l.note("set", ctx, err)
	return err
}

func (l *logged) SetNX(ctx context.Context, key string, v []byte, ttl time.Duration) (bool, error) {
	ok, err := l.CacheStore.SetNX(ctx, key, v, ttl)
	l.note("setnx", ctx, err)
	return ok, err
}

func (l *logged) Delete(ctx context.Context, keys ...string) error {
	err := l.CacheStore.Delete(ctx, keys...)
	l.note("delete", ctx, err)
	return err
}

// Options configures New.
type Options struct {
	// Name labels the primary in log messages; empty means "primary".
	Name string
	// Log receives primary errors (rate limited), degrade and recovery notices.
	Log Logger
	// LogEvery is the minimum gap between error warnings; 0 means DefaultLogEvery.
	LogEvery time.Duration
	// Retry is how long the primary is skipped after it fails; 0 means the Fallback's default.
	Retry time.Duration
	// MemoryEntries caps the in-memory secondary; 0 means 10000.
	MemoryEntries int
	// OnDegraded and OnRecovered are called on each transition, after the built-in log line. Use
	// them to publish a gauge or alert; they must not block.
	OnDegraded  func(err error)
	OnRecovered func()
}

// New composes primary (wrapped in Logged) with an in-memory secondary in a cachestore.Fallback.
// While the primary is down reads and writes use per-process memory, so entries and invalidations
// are not shared between replicas until it recovers; that is the right trade for a cache
// (degraded means "less sharing", not "unavailable"), and the Fallback's own doc explains it.
// Wrap the result in Bounded to keep a slow primary from stalling calls.
func New(primary cachestore.CacheStore, o Options) *cachestore.Fallback {
	name := o.Name
	if name == "" {
		name = "primary"
	}
	if o.MemoryEntries <= 0 {
		o.MemoryEntries = 10000
	}
	return cachestore.NewFallback(
		Logged(primary, name, o.Log, o.LogEvery),
		cachestore.NewMemoryStore(cachestore.MemoryOptions{MaxEntries: o.MemoryEntries}),
		cachestore.FallbackOptions{
			Retry: o.Retry,
			OnDegraded: func(err error) {
				o.Log.warn("cache: %s unavailable, serving from per-process memory (entries and invalidations are not shared between replicas): %v", name, err)
				if o.OnDegraded != nil {
					o.OnDegraded(err)
				}
			},
			OnRecovered: func() {
				o.Log.info("cache: %s recovered, shared cache restored", name)
				if o.OnRecovered != nil {
					o.OnRecovered()
				}
			},
		})
}
