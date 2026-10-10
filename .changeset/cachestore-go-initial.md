---
"@zenius-one/cachestore-go": minor
---

First release: `CacheStore` interface with `MemoryStore`, `RedisStore` and a `Fallback` that serves from memory while Redis is failing; `New` returns the MemoryStore alone when no Redis client is configured; `GetJSON`/`SetJSON` helpers.
