---
name: commit-strategy
description: Sequence commits so every one builds, and write changesets by hand. Use whenever staging or committing changes under packages/, SPEC.md, .changeset/ or .github/.
---

# Commit strategy

Read [`../../references/dependency-graph/README.md`](../../references/dependency-graph/README.md)
first: it is the map this skill operates on, and this file is the algorithm.

## Before committing

1. `git status` and `git diff` for the full set of changes.
2. Classify each file: the **contract** (`SPEC.md`), an **implementation** (`packages/*`), its
   **tests**, a **docs** file, or **repo plumbing** (`.github/`, `.changeset/`, root config).
3. A change to semantics in `SPEC.md` needs the matching implementation and conformance-test
   change. If an implementation changed behaviour without `SPEC.md` changing, either the spec was
   wrong or the implementation is; resolve that before committing.
4. Run, from the repo root: `pnpm turbo run build lint check-types test`. All four must pass.

## Ordering

1. **Never split a behaviour change from its tests.**
2. **Contract before implementations** when a change spans both and is split into commits.
3. **Upstream module before downstream.** A change to the core Go module lands (and is released)
   before the redis module pins it. Never commit a downstream use of an API whose upstream commit
   is not already in history.
4. **Plumbing changes stand alone**: workflow, script and config edits do not ride along with a
   behaviour change.
5. **Every commit builds and passes `pnpm turbo run build lint check-types test`.** If you cannot
   state the command that makes a commit's tree consistent, it is split wrong.

## Changesets

A changeset triggers a real release: version bump, changelog, tag and GitHub release on merge to
`main` (see `.github/README.md`). Write one for any change to a module listed in
`.github/publish-go-packages.config.json` (or `.github/publish-packages.config.json`).

- **A README-only change needs none**, and starts no workflow.
- Agents cannot drive the interactive `pnpm changeset` CLI (it needs a TTY): write the file by
  hand as `.changeset/<kebab-case-slug>.md`:

  ```markdown
  ---
  "cachestore-go": minor
  ---

  Add `Fallback.Degraded()` so callers can expose cache degradation as a metric.
  ```

  The name is the shim's `package.json` `name` (`cachestore-go`, `cachestore-go-redis`).
- **Bump type:** `patch` for fixes and internal changes with no new surface; `minor` for
  backward-compatible additions; `major` for anything that breaks an existing caller (changed
  semantics, removed or renamed API). While a module is `0.x`, breaking changes are `minor`, and
  the changeset text must say so.
- One changeset per logical change, listing every package it affects.
- After writing it by hand, run `node .github/scripts/sync-changeset-ignore.mjs --write` and
  commit any `.changeset/config.json` diff in the same commit. The root `changeset` script does
  this automatically, but only when a human runs it.
- Verify: `pnpm exec changeset status --since=origin/main`.

## Changing the core module when redis depends on it

Release core first, then, as a follow-up change, pin the new core version in
`packages/cachestore-go/redis` with
`GOWORK=off go get github.com/zenius-one/cachestore/packages/cachestore-go@vX.Y.Z` and add a
changeset for `cachestore-go-redis` if its behaviour or requirement changed. Never pin a version
that has not been tagged.
