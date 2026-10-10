# cachestore-go/redis

```go
import cacheredis "github.com/zenius-one/cachestore/packages/cachestore-go/redis"
```

The Redis implementation of `cachestore.CacheStore`, in its own module so the core module (and
every service that only needs a memory cache) stays free of a Redis client and its dependencies.
Behaviour contract: [`SPEC.md`](../../../SPEC.md).

```go
// client may be nil (no Redis configured): you get the MemoryStore alone.
store := cacheredis.New(redisClient, "myservice:cache:", cachestore.MemoryOptions{MaxEntries: 10000})
defer store.Close()
```

With a client, `New` returns a `cachestore.Fallback`: Redis while it works; after an error it is
skipped for 5 seconds and memory serves instead, recovering automatically. Use
`cachestore.FallbackOptions` (third argument) for `OnDegraded` / `OnRecovered` hooks and the retry
window.

`redisClient` is any `redis.Cmdable` from `github.com/redis/go-redis/v9` (`*redis.Client`,
`*redis.ClusterClient`, ...). The client belongs to the caller; `Close` does not close it. Every
key is stored under the prefix, so several users of one Redis cannot collide and a prefix can be
flushed on its own.

`NewRedisStore(client, prefix)` gives the bare Redis store when you want to compose it yourself.

## Development

```sh
go vet ./... && go test -race ./...
```

Redis is faked with `miniredis`. Inside the repository the root `go.work` resolves the core
module locally.
