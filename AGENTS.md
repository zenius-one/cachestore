<!-- BEGIN:turborepo-agent-rules -->

# This is NOT the Turborepo you know

Turborepo configuration, task behavior, and CLI commands can vary between installed versions and may differ from your training data. Resolve the `turbo` package from this file's directory or relevant workspace; in monorepos, it may not be visible from the repository root. For example, run `node -p "require.resolve('turbo/package.json')"` from a workspace that depends on `turbo`.

Read `docs/README.md` inside that installed package first, then read the relevant pages from its `docs/` directory before changing Turborepo configuration or commands. Heed deprecation notices. These bundled docs match the installed package version and are available without network access.

This block is written and re-added by `turbo` before repository-scoped commands when an AI agent is detected. In the Turborepo source repository, its template is defined in `crates/turborepo-cli/src/cli/agent_guidance.rs`. Removing the managed block while updates are enabled means a later qualifying invocation will add it again. Set `"agentGuidance": false` in the root `turbo.json` or `turbo.jsonc` to opt out; this does not remove an existing block. Keep the block committed with your work to avoid an uncommitted change on the next agent invocation.
<!-- END:turborepo-agent-rules -->

# Repository instructions

This repository's process lives in [`.ai/`](.ai/README.md); read it before changing anything.

- Committing and changesets: [`.ai/skills/commit-strategy/SKILL.md`](.ai/skills/commit-strategy/SKILL.md)
- Issues, branches and PRs: [`.ai/skills/pr-strategy/SKILL.md`](.ai/skills/pr-strategy/SKILL.md)
- Dependency map: [`.ai/references/dependency-graph/README.md`](.ai/references/dependency-graph/README.md)
- Behaviour contract: [`SPEC.md`](SPEC.md)

Core rules: one issue per PR; behaviour changes start in `SPEC.md`; the core Go module stays free
of third-party dependencies; a module change needs a hand-written changeset; run
`pnpm turbo run build lint check-types test`.
