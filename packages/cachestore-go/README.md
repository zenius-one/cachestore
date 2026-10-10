# cachestore (Go)

```go
import "github.com/zenius-one/cachestore/packages/cachestore-go"
```

Package `cachestore`. The behaviour contract is [`../SPEC.md`](../../SPEC.md).

```go
// client may be nil (no Redis configured): you get the MemoryStore alone.
store := cachestore.New(redisClient, "myservice:cache:", cachestore.MemoryOptions{MaxEntries: 10000})
defer store.Close()

if v, ok, err := store.Get(ctx, key); err == nil && ok {
    // hit
}
_ = store.Set(ctx, key, value, 30*time.Second)

// typed
_ = cachestore.SetJSON(ctx, store, key, thing, time.Minute)
thing, ok, _ := cachestore.GetJSON[Thing](ctx, store, key)
```

`redisClient` is any `redis.Cmdable` from `github.com/redis/go-redis/v9` (`*redis.Client`,
`*redis.ClusterClient`, ...). The client belongs to the caller; `Close` does not close it.

Observe degradation with `FallbackOptions.OnDegraded` / `OnRecovered`, or `Fallback.Degraded()`:

```go
f := cachestore.NewFallback(
    cachestore.NewRedisStore(client, "svc:"),
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
