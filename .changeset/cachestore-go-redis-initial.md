---
"cachestore-go-redis": minor
---

First release: `RedisStore` (a `cachestore.CacheStore` over any `redis.Cmdable`, keys under a prefix) and `New`, which returns the memory store alone when no Redis client is configured and otherwise a `cachestore.Fallback` that serves from memory while Redis is failing. Requires `cachestore-go` v0.1.0.
