# CI and release

Same structure as iam-360 and nexthis: Changesets decides what changed, one workflow publishes it.

| Workflow | Runs on | Does |
| --- | --- | --- |
| `ci.yml` | PRs and `main`, except markdown-only changes | `turbo run build lint check-types test` |
| `changeset-checks.yml` | PRs touching `packages/**` or `.changeset/**`, except READMEs | requires a changeset; checks the changeset ignore list and that no Go module shim was deleted or regressed |
| `publish-packages.yml` | pushes to `main` touching `.changeset/**` or the publish configs; manual | version, then publish npm packages (GitHub Packages) and tag + release Go modules |

## How a Go module is released

Each Go module has a `package.json` next to its `go.mod` that is a **version-tracking shim**: never
published to npm, it exists so Changesets can bump the version and write `CHANGELOG.md`, and so
Turborepo can run its scripts. `publish-packages.yml`:

1. `changeset version` consumes pending changesets and commits the bump to `main` (`[skip ci]`).
2. For each bumped module listed in `publish-go-packages.config.json`, a matrix job builds and
   vets it with `GOWORK=off` (as a consumer sees it), tags `packages/<dir>/vX.Y.Z` and creates a
   GitHub release from the changelog section with GoReleaser.

The tag is prefixed with the module's directory because that is how the Go toolchain resolves a
module in a subdirectory.

## Recovering a missed publish

A failed publish job leaves the version bumped in git but untagged, and nothing under
`.changeset/**` changed, so a push will not retrigger. Run **Publish Packages** manually with
`mode: reconcile`: it skips changeset consumption and republishes whatever is committed but
missing from the registry or tags.

## Eligibility

- `publish-packages.config.json` (npm) and `publish-go-packages.config.json` (Go) are the only
  places that say what may be published; `.changeset/config.json`'s `ignore` list is generated
  from them (`pnpm changeset` re-syncs it, the checks workflow verifies it).
- **`packages/cachestore-go/redis` is not listed yet.** Its `go.mod` has no requirement on the core
  module until `packages/cachestore-go/v0.1.0` is tagged. Then: pin it with
  `GOWORK=off go get github.com/zenius-one/cachestore/packages/cachestore-go@v0.1.0`, add the
  module to `publish-go-packages.config.json`, and add a changeset.

## Repository settings this cannot apply itself

- `main`'s branch protection must allow `github-actions[bot]` to push the version commit.
- Secrets: `NPM_TOKEN` (org PAT with `read:packages`) only matters once an npm package depends on
  private packages from other repos.

## README-only changes

`**/README.md` is excluded from `changeset-checks.yml`'s paths and CI ignores `**/*.md`, so a
README-only PR starts no workflow. If `Changeset Checks` is a required status check, GitHub shows
it as "expected" on such PRs.
