# Copilot instructions

This repository is a pnpm + Turborepo + Changesets monorepo of cache-abstraction implementations
(`CacheStore`, `MemoryStore`, a Redis store, a Redis-to-memory fallback), one package per language.
`SPEC.md` is the behaviour contract. Process lives in `.ai/`; do not duplicate it here.

## Layout

- `SPEC.md`: language-neutral contract and conformance cases.
- `packages/cachestore-go/`: Go core module. **No third-party dependencies.**
- `packages/cachestore-go/redis/`: separate Go module with the Redis store.
- Each Go module has a `package.json` shim (version tracking and Turborepo scripts only).

## Always

- Start behaviour changes in `SPEC.md` and update every implementation and its conformance tests.
- Keep the core Go module dependency-free; Redis code belongs in `packages/cachestore-go/redis`.
- Treat cache errors as misses; never let a cache failure fail a request; hash anything secret
  before using it in a key; require a positive TTL on every write.
- Add a changeset (`.changeset/<slug>.md`, package name from the shim's `package.json`) for any
  change to a published module. A README-only change needs none.
- Run `pnpm turbo run build lint check-types test` before proposing a change.
- Use conventional commit subjects (`feat(cachestore-go): ...`).
- One issue per PR; branch `<type>/<issue-number>-short-description`; exactly one `Closes #n`;
  fill every section of `.github/PULL_REQUEST_TEMPLATE.md`.

## Never

- Hand-edit `.changeset/config.json`'s `ignore` list (generated).
- Add a core-module dependency, or commit a release tag or version bump by hand.
- Push to `main`.

Procedures: `.ai/skills/commit-strategy/SKILL.md`, `.ai/skills/pr-strategy/SKILL.md`.
Dependency map: `.ai/references/dependency-graph/README.md`.
