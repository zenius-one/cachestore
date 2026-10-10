// Package cachestore is the one place caching goes through, so a service never hand-rolls a map
// with a mutex or talks to Redis directly for cache purposes.
//
//   - CacheStore is the interface callers depend on.
//   - MemoryStore is a bounded in-process implementation.
//   - RedisStore is the distributed implementation.
//   - Fallback composes them: with Redis present it is the primary and degrades to a MemoryStore
//     while Redis is failing; with no Redis configured, New returns the MemoryStore alone.
//
// Values are opaque bytes; use GetJSON/SetJSON for typed values. Keys must be tenant/caller scoped
// by the caller: this package does not know what they mean. Never put a secret in a key (hash it),
// and remember that a shared store makes entries visible to every replica, which is the point for
// introspection results and the hazard for anything caller-specific.
package cachestore
