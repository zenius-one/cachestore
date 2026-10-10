# Dependency graph

The map `skills/commit-strategy` operates on. Update it whenever a package is added, moved or
gains a dependency.

## Behaviour contract

`SPEC.md` is the source of truth for behaviour. Every language implementation is derived from it
by hand (there is no code generation), so a change to semantics is a change to `SPEC.md` **and**
to every implementation, with matching conformance tests.

```
SPEC.md
  |-- packages/cachestore-go          (Go core: interface, MemoryStore, Fallback, JSON helpers)
  |     `-- packages/cachestore-go/redis   (Go: RedisStore + New; requires the core module)
  `-- packages/cachestore-ts          (planned)
```

## Go modules

| Module | Path | Requires |
| --- | --- | --- |
| core | `packages/cachestore-go` | nothing (no third-party dependencies, deliberately) |
| redis | `packages/cachestore-go/redis` | core, `github.com/redis/go-redis/v9` |

- A dependency edge means: change the upstream first, release it, pin the new version downstream.
- The root `go.work` resolves both modules locally. `GOWORK=off` is how a consumer sees them and
  how CI builds before tagging; it needs the required version to exist as a tag.
- The core module must stay free of third-party dependencies: that is why Redis is a separate
  module. A change that adds a dependency to core is wrong, whatever the reason.
- The redis module's `go.mod` omits the core requirement until `packages/cachestore-go/v0.1.0`
  exists (Go will not resolve a required version that has no tag, even in a workspace).

## Release graph

Each Go module has a `package.json` next to its `go.mod`. It is a **version-tracking shim** (never
published to npm) so Changesets can bump it and Turborepo can run its scripts.

| Shim name | Module | Released when |
| --- | --- | --- |
| `cachestore-go` | core | listed in `.github/publish-go-packages.config.json` and bumped by a changeset |
| `cachestore-go-redis` | redis | not yet listed; listed after core `v0.1.0` is tagged and pinned |

Tags are `packages/<dir>/vX.Y.Z` (`packages/cachestore-go/redis/vX.Y.Z` for the nested module):
the Go toolchain resolves a module in a subdirectory by its directory-prefixed tag.

Eligibility lives only in `.github/publish-packages.config.json` and
`.github/publish-go-packages.config.json`; `.changeset/config.json`'s `ignore` list is generated
from them by `.github/scripts/sync-changeset-ignore.mjs`.
