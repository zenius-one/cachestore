# cachestore (Go)

```go
import "github.com/zenius-one/cachestore/packages/cachestore-go"
```

Package `cachestore`. The behaviour contract is [`SPEC.md`](../../SPEC.md).

```go
store := cachestore.NewMemoryStore(cachestore.MemoryOptions{MaxEntries: 10000})
defer store.Close()

if v, ok, err := store.Get(ctx, key); err == nil && ok {
    // hit
}
_ = store.Set(ctx, key, value, 30*time.Second)

// typed
_ = cachestore.SetJSON(ctx, store, key, thing, time.Minute)
thing, ok, _ := cachestore.GetJSON[Thing](ctx, store, key)
```

The core module has no third-party dependencies. For a distributed cache with a memory fallback use
the [`redis`](redis) module, whose `New` picks the right composition.

Observe degradation with `FallbackOptions.OnDegraded` / `OnRecovered`, or `Fallback.Degraded()`:

```go
f := cachestore.NewFallback(
    primary, // any cachestore.CacheStore
    cachestore.NewMemoryStore(cachestore.MemoryOptions{}),
    cachestore.FallbackOptions{OnDegraded: func(err error) { log.Warn("cache degraded", "err", err) }},
)
```

## Development

```sh
go vet ./... && go test -race ./...
# or, from the repo root:
pnpm turbo run build lint check-types test
```

Redis is faked with `miniredis`; no external services are needed.
