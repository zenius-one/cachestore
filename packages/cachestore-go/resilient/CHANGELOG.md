# cachestore-go-resilient

## 0.2.0

### Minor Changes

- [#9](https://github.com/zenius-one/cachestore/pull/9) [`ca0eb21`](https://github.com/zenius-one/cachestore/commit/ca0eb218f1d9445c8b3957c4a4940af6c57ae65a) Thanks [@jayantasamaddar](https://github.com/jayantasamaddar)! - Add the `resilient` module: `Bounded` (per-operation timeout), `Logged` (rate-limited error logging), `New` (logged primary + in-memory Fallback with transition hooks) and `Prober` (recovery detection for callers that skip the cache while degraded). No third-party dependencies.
