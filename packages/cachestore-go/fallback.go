package cachestore

import (
	"context"
	"sync"
	"time"
)

// FallbackOptions configures a Fallback.
type FallbackOptions struct {
	// Retry is how long the primary is skipped after it fails; 0 means 5 seconds. It stops every
	// request paying the primary's timeout during an outage.
	Retry time.Duration
	// OnDegraded and OnRecovered are optional hooks, called on each transition (for logs/metrics).
	OnDegraded  func(err error)
	OnRecovered func()
	// Now is overridable for tests.
	Now func() time.Time
}

// Fallback is a CacheStore that uses primary while it works and secondary while it does not.
//
// While the primary is down, reads and writes go to the secondary only, so entries written then
// are per-process and are not visible to other replicas; after recovery they are simply not
// consulted (the primary is the source of truth), and they expire on their own TTL. That is the
// right trade for a cache: degraded means "less sharing", never "wrong" or "unavailable".
type Fallback struct {
	primary, secondary CacheStore
	opts               FallbackOptions

	mu        sync.Mutex
	downUntil time.Time
	degraded  bool
}

var _ CacheStore = (*Fallback)(nil)

// NewFallback composes primary and secondary.
func NewFallback(primary, secondary CacheStore, opts FallbackOptions) *Fallback {
	if opts.Retry <= 0 {
		opts.Retry = 5 * time.Second
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Fallback{primary: primary, secondary: secondary, opts: opts}
}

// usePrimary reports whether the primary should be tried now.
func (f *Fallback) usePrimary() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.opts.Now().Before(f.downUntil)
}

func (f *Fallback) failed(err error) {
	f.mu.Lock()
	f.downUntil = f.opts.Now().Add(f.opts.Retry)
	was := f.degraded
	f.degraded = true
	f.mu.Unlock()
	if !was && f.opts.OnDegraded != nil {
		f.opts.OnDegraded(err)
	}
}

func (f *Fallback) succeeded() {
	f.mu.Lock()
	was := f.degraded
	f.degraded = false
	f.mu.Unlock()
	if was && f.opts.OnRecovered != nil {
		f.opts.OnRecovered()
	}
}

// Degraded reports whether the primary is currently being skipped.
func (f *Fallback) Degraded() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.degraded
}

// Get implements CacheStore.
func (f *Fallback) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if f.usePrimary() {
		v, ok, err := f.primary.Get(ctx, key)
		if err == nil {
			f.succeeded()
			return v, ok, nil
		}
		f.failed(err)
	}
	return f.secondary.Get(ctx, key)
}

// Set implements CacheStore.
func (f *Fallback) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	if f.usePrimary() {
		err := f.primary.Set(ctx, key, value, ttl)
		if err == nil {
			f.succeeded()
			return nil
		}
		f.failed(err)
	}
	return f.secondary.Set(ctx, key, value, ttl)
}

// SetNX implements CacheStore. While degraded the lease is process-local: it still collapses
// requests within one replica, which is the best available.
func (f *Fallback) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}
	if f.usePrimary() {
		ok, err := f.primary.SetNX(ctx, key, value, ttl)
		if err == nil {
			f.succeeded()
			return ok, nil
		}
		f.failed(err)
	}
	return f.secondary.SetNX(ctx, key, value, ttl)
}

// Delete implements CacheStore. It always clears the secondary too, so a deletion made while the
// primary was down cannot resurrect through a stale local entry.
func (f *Fallback) Delete(ctx context.Context, keys ...string) error {
	secErr := f.secondary.Delete(ctx, keys...)
	if f.usePrimary() {
		err := f.primary.Delete(ctx, keys...)
		if err == nil {
			f.succeeded()
			return secErr
		}
		f.failed(err)
	}
	return secErr
}

// Close implements CacheStore, closing both stores.
func (f *Fallback) Close() error {
	perr := f.primary.Close()
	serr := f.secondary.Close()
	if perr != nil {
		return perr
	}
	return serr
}
