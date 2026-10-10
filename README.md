# cachestore

One cache abstraction for every service, in every language you write services in. A service
depends on the `CacheStore` interface, never on a map with a mutex or on Redis directly; the
backing store is a deployment decision.

<!--prettier-ignore-->
| Language | Directory | Module / package | Status |
| --- | --- | --- | --- |
| Go | [`packages/cachestore-go`](packages/cachestore-go) | `github.com/zenius-one/cachestore/packages/cachestore-go` | implemented |
| Go, Redis store | [`packages/cachestore-go/redis`](packages/cachestore-go/redis) | `github.com/zenius-one/cachestore/packages/cachestore-go/redis` | implemented |
| TypeScript | `packages/cachestore-ts` | `cachestore` | planned |

The contract every implementation follows is [`SPEC.md`](SPEC.md). Implementations differ in
idiom, not in behaviour: a service moving between languages keeps its caching semantics.

## Stores

| Store         |                                                                                         |
| ------------- | --------------------------------------------------------------------------------------- |
| `MemoryStore` | bounded, in-process (per replica); in the core module                                   |
| `Fallback`    | composes any two stores: serves from the secondary while the primary fails; core module |
| `RedisStore`  | distributed, shared by every replica, keys under a prefix; **separate module**          |

Redis is a separate module on purpose: a service that never uses it never depends on a Redis
client, so it never reaches that service's build or bundle. The Redis module's `New(redisClient,
prefix, memoryOptions)` returns the `MemoryStore` alone when no Redis client is configured, and a
`Fallback` over Redis and memory when one is. Degraded means "less sharing between replicas",
never "wrong" and never "unavailable".

## Layout and releases

A pnpm + Turborepo + Changesets monorepo, one package per language under
`packages/`, each independently versioned and released.

```
packages/cachestore-go/    Go core module (interface, MemoryStore, Fallback); its package.json is a
                           version-tracking shim whose scripts (build, lint, check-types, test)
                           call the Go toolchain so Turborepo runs it like any package
packages/cachestore-go/redis/   Go module for the Redis store; depends on the core module
.changeset/                pending changesets and config
.github/workflows/         ci, changeset-checks, release
```

- **Develop:** `pnpm install`, then `pnpm turbo run build lint check-types test`.
- **Change a package:** `pnpm changeset` (a README-only change needs none).
- **Release:** `publish-packages.yml` on `main`: `changeset version` bumps and writes the
  changelog, then each module listed in `.github/publish-go-packages.config.json` gets
  `packages/<dir>/vX.Y.Z` (`packages/cachestore-go/redis/vX.Y.Z` for the nested module) and a
  GitHub release. `mode: reconcile` recovers a failed run. See [`.github/README.md`](.github/README.md).
- **CI** ignores markdown-only changes.

## Adding a language

1. Create `packages/cachestore-<lang>/` with the idiomatic layout, and a `package.json` whose
   `build`, `lint`, `check-types` and `test` scripts run that ecosystem's tooling.
2. Implement the interface and the three stores to [`SPEC.md`](SPEC.md), including the conformance
   cases listed there.
3. Add its row above and, if it publishes somewhere other than a git tag, its publish step to
   `release.yml`.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). Process for humans and AI assistants lives in [`.ai/`](.ai/README.md).
