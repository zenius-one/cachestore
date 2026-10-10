# cachestore-go-redis

## 0.1.0

### Minor Changes

- [#4](https://github.com/zenius-one/cachestore/pull/4) [`6a53c26`](https://github.com/zenius-one/cachestore/commit/6a53c260b181a6950227f983ad097037d449be36) Thanks [@jayantasamaddar](https://github.com/jayantasamaddar)! - First release: `RedisStore` (a `cachestore.CacheStore` over any `redis.Cmdable`, keys under a prefix) and `New`, which returns the memory store alone when no Redis client is configured and otherwise a `cachestore.Fallback` that serves from memory while Redis is failing. Requires `cachestore-go` v0.1.0.
