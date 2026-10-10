# cachestore-go

## 0.1.0

### Minor Changes

- [#2](https://github.com/zenius-one/cachestore/pull/2) [`8fe80ed`](https://github.com/zenius-one/cachestore/commit/8fe80ed745c01a9fcc7f160b612f84c4e7494372) Thanks [@jayantasamaddar](https://github.com/jayantasamaddar)! - First release: `CacheStore` interface, `MemoryStore`, a `Fallback` that composes any two stores (serving from the secondary while the primary fails), and `GetJSON`/`SetJSON` helpers. No third-party dependencies; the Redis store is the separate `packages/cachestore-go/redis` module.
