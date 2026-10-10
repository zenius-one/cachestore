---
"cachestore-go": minor
---

First release: `CacheStore` interface, `MemoryStore`, a `Fallback` that composes any two stores (serving from the secondary while the primary fails), and `GetJSON`/`SetJSON` helpers. No third-party dependencies; the Redis store is the separate `packages/cachestore-go/redis` module.
