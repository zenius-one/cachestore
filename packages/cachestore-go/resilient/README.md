# cachestore-go/resilient

Wrappers that make a [`CacheStore`](../README.md) degrade quickly and visibly when its backend
(normally Redis) is slow or down. No third-party dependencies; requires only the core module.

| Function | What it does |
| --- | --- |
| `Bounded(store, timeout)` | each operation gets its own deadline (the caller's context still wins if shorter) |
| `Logged(store, name, log, every)` | logs the errors a `Fallback` swallows, at most one warning per `every`, counting the suppressed ones |
| `New(primary, Options)` | `Logged(primary)` + in-memory secondary in a `cachestore.Fallback`, with log lines and `OnDegraded` / `OnRecovered` hooks |
| `NewProber(fb, ...)` | `Active()` / `Probe()` for callers that skip the cache while degraded and still need to notice recovery |

```go
fb := resilient.New(redisStore, resilient.Options{
    Name: "redis", Log: resilient.Logger{Warnf: log.Warnf, Infof: log.Infof},
    OnDegraded:  func(error) { breakerGauge.Set(1) },
    OnRecovered: func() { breakerGauge.Set(0) },
})
store := resilient.Bounded(fb, 300*time.Millisecond) // use this as your CacheStore
probe := resilient.NewProber(fb, 0, 0)               // only if some data must not be served per-process
```

Hooks are plain callbacks on purpose: this module does not import Prometheus, a logger or any
cache-above-the-store library. Policy (for example "this data must not be stale, so bypass the
cache while `probe.Active()`") belongs to the caller.
