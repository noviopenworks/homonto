## Why

Homonto's config model parses `[frameworks.X]`, `[commands.X]`, and `[subagents.X]` with `source = "builtin:<name>"`, but current adapters only project local-source skills. The first-release bundled frameworks (`onto`, `comet`, `superpowers`, `openspec`) cannot be installed today. This change adds the catalog foundation: a bundled, embedded catalog; framework metadata with dependency expansion; and builtin skill projection -- the base layer that command and subagent projection build on later.

## What Changes

- Add a bundled catalog directory (`catalog/`) at the repo root, embedded in the binary via `go:embed`.
- Add framework metadata files (`catalog/frameworks/<name>/framework.toml`) declaring name, version, dependencies, and bundled skill resources.
- Implement framework expansion: `[frameworks.X] source = "builtin:<name>"` expands to its constituent skills plus transitive dependency skills.
- Implement builtin source resolution for skills: `source = "builtin:<name>"` materializes from the embedded catalog to `.homonto/catalog/skills/<name>/` and symlinks from there.
- Extend both Claude Code and OpenCode adapters to handle builtin-source skills alongside existing local-source skills.
- Populate the catalog with all four first-release frameworks: `onto`, `comet`, `superpowers`, `openspec`.
- Add state tracking for materialized catalog resources, including version-aware re-materialization.

## Capabilities

### New Capabilities

- `builtin-catalog`: Bundled catalog structure, go:embed integration, materialization to `.homonto/catalog/`, and builtin source resolution for skills.
- `framework-expansion`: Framework metadata format, dependency expansion, transitive resolution, and atomic framework enable/disable semantics.

### Modified Capabilities

- `config-model`: Frameworks and skills now have projection behavior for builtin sources, not just validation.
- `tool-adapters`: Adapters resolve and project builtin-source skills from the materialized catalog path.

## Impact

- New `catalog/` directory tree with framework metadata and bundled skill content.
- New `internal/catalog/` Go package for catalog loading, framework expansion, and materialization.
- Modified `internal/config/config.go` for framework expansion hooks.
- Modified `internal/adapter/{claude,opencode}` for builtin source resolution.
- Modified `internal/engine/` for materialization orchestration.
- New tests for catalog parsing, framework expansion, materialization, and builtin skill projection.
- `homonto.toml` may use `[frameworks.comet]` instead of individual `[skills.comet]` entries.
