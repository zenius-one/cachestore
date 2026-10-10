# cachestore

One cache abstraction for every Zenius service, in every language we write services in. A service
depends on the `CacheStore` interface, never on a map with a mutex or on Redis directly; the
backing store is a deployment decision.

| Language | Directory | Module / package | Status |
| --- | --- | --- | --- |
| Go | [`go/`](go) | `github.com/zenius-one/cachestore/go` | implemented |
| TypeScript | `ts/` | `@zenius-one/cachestore` | planned |

The contract every implementation follows is [`SPEC.md`](SPEC.md). Implementations differ in
idiom, not in behaviour: a service moving between languages keeps its caching semantics.

## Stores

| Store | |
| --- | --- |
| `MemoryStore` | bounded, in-process (per replica) |
| `RedisStore` | distributed, shared by every replica, keys under a prefix |
| `Fallback` | Redis primary that degrades to a `MemoryStore` while Redis fails |

`New(redisClient, prefix, memoryOptions)` returns the `MemoryStore` alone when no Redis client is
configured, and a `Fallback` over Redis and memory when one is. Degraded means "less sharing
between replicas", never "wrong" and never "unavailable".

## Layout and releases

One top-level directory per language, each independently versioned and released; nothing outside
a language directory may be imported by another. Go tags are `go/vX.Y.Z` (the Go toolchain's
convention for a module in a subdirectory); other languages follow their own registry's
convention with the language prefix, e.g. `ts/vX.Y.Z`.

CI runs per language and only when that language's directory changes. Markdown-only changes start
no workflow.

## Adding a language

1. Create `<lang>/` with the idiomatic layout for that ecosystem.
2. Implement the interface and the three stores to [`SPEC.md`](SPEC.md), including the conformance
   cases listed there.
3. Add its row above and a path-filtered workflow under `.github/workflows/`.
