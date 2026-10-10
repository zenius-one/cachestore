# cachestore

One cache abstraction for every Zenius service, in every language we write services in. A service
depends on the `CacheStore` interface, never on a map with a mutex or on Redis directly; the
backing store is a deployment decision.

| Language | Directory | Module / package | Status |
| --- | --- | --- | --- |
| Go | [`packages/cachestore-go`](packages/cachestore-go) | `github.com/zenius-one/cachestore/packages/cachestore-go` | implemented |
| TypeScript | `packages/cachestore-ts` | `@zenius-one/cachestore` | planned |

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

A pnpm + Turborepo + Changesets monorepo, same shape as iam-360: one package per language under
`packages/`, each independently versioned and released.

```
packages/cachestore-go/    Go module; its package.json is a version-tracking shim whose scripts
                           (build, lint, check-types, test) call the Go toolchain so Turborepo
                           runs it like any package
.changeset/                pending changesets and config
.github/workflows/         ci, changeset-checks, release
```

- **Develop:** `pnpm install`, then `pnpm turbo run build lint check-types test`.
- **Change a package:** `pnpm changeset` (a README-only change needs none).
- **Release:** merging to `main` runs `release.yml`: `changeset version` bumps and writes the
  changelog, then every Go module whose committed version has no tag gets
  `packages/<dir>/vX.Y.Z` and a GitHub release. Re-running the workflow recovers a failed run.
  The tag is prefixed with the module's directory because that is how the Go toolchain resolves a
  module that lives in a subdirectory.
- **CI** ignores markdown-only changes.

## Adding a language

1. Create `packages/cachestore-<lang>/` with the idiomatic layout, and a `package.json` whose
   `build`, `lint`, `check-types` and `test` scripts run that ecosystem's tooling.
2. Implement the interface and the three stores to [`SPEC.md`](SPEC.md), including the conformance
   cases listed there.
3. Add its row above and, if it publishes somewhere other than a git tag, its publish step to
   `release.yml`.
