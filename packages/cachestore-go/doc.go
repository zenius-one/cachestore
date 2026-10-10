// Package cachestore is the one place caching goes through, so a service never hand-rolls a map
// with a mutex or talks to Redis directly for cache purposes.
//
//   - CacheStore is the interface callers depend on.
//   - MemoryStore is a bounded in-process implementation.
//   - Fallback composes any two stores: the primary is used while it works and the secondary
//     while it does not.
//   - The distributed implementation, RedisStore, lives in the separate module
//     github.com/zenius-one/cachestore/packages/cachestore-go-redis, so a service that never uses
//     Redis never depends on a Redis client. That module also has New, which picks Redis with a
//     memory fallback, or memory alone when no Redis is configured.
//
// Values are opaque bytes; use GetJSON/SetJSON for typed values. Keys must be tenant/caller scoped
// by the caller: this package does not know what they mean. Never put a secret in a key (hash it),
// and remember that a shared store makes entries visible to every replica, which is the point for
// introspection results and the hazard for anything caller-specific.
package cachestore
