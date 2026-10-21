// Package resilient wraps a cachestore.CacheStore so that a slow or dead backend (normally Redis)
// degrades a cache quickly and visibly instead of stalling requests or failing silently.
//
// It has no third-party dependencies and knows nothing about what sits above the store: hooks are
// plain callbacks, so metrics, logging and policy stay with the caller.
//
//   - Bounded gives every store operation its own short deadline.
//   - Logged reports the errors a Fallback swallows, rate limited.
//   - New composes the two with an in-memory Fallback and transition hooks.
//   - Prober lets a caller that skips the cache while degraded notice the recovery.
//
// Typical composition, outermost first:
//
//	store := resilient.Bounded(resilient.New(redisStore, resilient.Options{...}).Fallback, 300*time.Millisecond)
package resilient
