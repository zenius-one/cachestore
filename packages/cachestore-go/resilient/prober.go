package resilient

import (
	"context"
	"sync/atomic"
	"time"

	cachestore "github.com/zenius-one/cachestore/packages/cachestore-go"
)

// DefaultProbeEvery is how often, while the primary is down, Prober.Probe checks whether it is back.
const DefaultProbeEvery = 5 * time.Second

// probeKey is read purely to give the Fallback a chance to try its primary again; it never exists.
const probeKey = "resilient:probe"

// Prober answers "is the primary down?" for callers that must NOT use the per-process fallback
// (for example data that must never be served stale), and detects recovery for them. A Fallback
// retries its primary only when an operation reaches it; a caller that bypasses the cache while
// degraded issues none, so without a probe it would stay bypassed forever.
type Prober struct {
	fb      *cachestore.Fallback
	every   time.Duration
	timeout time.Duration
	now     func() time.Time
	last    atomic.Int64 // unix nanos of the last probe
}

// NewProber watches fb. every <= 0 means DefaultProbeEvery; timeout bounds one probe (0: 300ms).
func NewProber(fb *cachestore.Fallback, every, timeout time.Duration) *Prober {
	if every <= 0 {
		every = DefaultProbeEvery
	}
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}
	return &Prober{fb: fb, every: every, timeout: timeout, now: time.Now}
}

// Active reports whether the primary is currently down. Safe on a nil receiver.
func (p *Prober) Active() bool { return p != nil && p.fb.Degraded() }

// Probe lets the Fallback try its primary again, at most once per interval, off the caller's path.
// Safe on a nil receiver.
func (p *Prober) Probe() {
	if p == nil {
		return
	}
	now := p.now().UnixNano()
	last := p.last.Load()
	if now-last < int64(p.every) || !p.last.CompareAndSwap(last, now) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		defer cancel()
		_, _, _ = p.fb.Get(ctx, probeKey)
	}()
}
