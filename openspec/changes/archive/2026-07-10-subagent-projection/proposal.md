## Why

Homonto's config model parses `[subagents.X]` resources and validates that any
tool a subagent targets has all three model levels defined, but nothing projects
them — subagents are the last declared resource kind that is parsed and then
ignored at apply. The archived `catalog-foundation-skills` and
`command-projection` changes built and proved the reusable projection foundation
(embedded catalog, version-gated materialization, managed-root symlinking,
scope-aware placement, adopt/prune, doctor verification). This change reuses that
foundation to project subagents into Claude Code and OpenCode, closing the v1.1
catalog projection surface. Unlike `command-projection`, it ships **real bundled
subagent content**, not just a placeholder fixture.

## What Changes

- Add builtin/local **subagent** projection: a `[subagents.<name>]` resource
  (`source = "builtin:<name>"` or `"local:<name>"`, with a required `scope`)
  links a single `.md` file into Claude Code (`~/.claude/agents/<name>.md`,
  project `<repo>/.claude/agents/<name>.md`) and OpenCode
  (`~/.config/opencode/agent/<name>.md`, project
  `<repo>/.opencode/agent/<name>.md`), scope-aware like commands. Note the
  per-tool directory names: Claude uses `agents/` (plural), OpenCode uses
  `agent/` (singular), mirroring OpenCode's singular `command/`.
- Materialize subagents **verbatim**: the projected `.md` is byte-for-byte the
  catalog/local source (symlinked). Model routing is **not** injected into
  subagent frontmatter; the existing `[models.<tool>.<level>]` validation stays
  as-is as a guard.
- Add a `catalog/subagents/<name>.md` area to the embedded catalog and
  **single-file** materialization to `.homonto/catalog/subagents/<name>.md`
  (reusing the command single-file pattern, not the skill directory pattern).
- Extend `framework.toml` with an optional `[subagents]` table and expand
  framework-declared subagents through `[frameworks.X]`, transitively,
  mirroring command expansion.
- Extend both adapters and `doctor` to plan/apply/adopt/prune/verify subagent
  links, reusing the managed-root and version-gated materialization foundation.
- Ship **three real bundled subagents**: `code-reviewer` and `codebase-explorer`
  as loose builtin subagents (framework-agnostic), plus one comet-framework
  subagent declared in the `comet` framework's `[subagents]` table so
  framework-declared subagent expansion is exercised with real content.
- Subagents are **flat** only in this change (`subagents/<name>.md` →
  `<name>`); namespaced subagents and per-subagent model overrides are non-goals.

## Capabilities

### New Capabilities

- `subagent-projection`: builtin/local subagent source resolution, single-file
  verbatim materialization from the embedded catalog, projection into Claude Code
  and OpenCode agent directories with conflict-safe managed-root linking,
  adoption, and pruning, framework `[subagents]` expansion, doctor verification
  of subagent links, and the three bundled real subagents.

### Modified Capabilities

- `config-model`: `[subagents.X]` gains projection behavior (materialize +
  link), not just parse/validate; the claim that subagent resolution is future
  work no longer holds.
- `framework-expansion`: the framework metadata format's `[subagents]` table
  becomes an expanded resource kind (previously reserved as "later").

## Impact

- New `catalog/subagents/` tree (three real subagents) embedded via `go:embed`.
- New `internal/subagentpath` (or extended `commandpath`/`skillpath`) mapping
  `(tool, scope)` to an agent directory, accounting for OpenCode's singular
  `agent/`.
- Modified `internal/catalog` (single-file subagent materialization +
  `ExpandSubagents`), `internal/config` (`ExpandedSubagentEntriesForTool` +
  framework `[subagents]` expansion), `internal/engine` (materialize subagents
  alongside skills/commands + `WithSubagentCatalogRoot`),
  `internal/adapter/{claude,opencode}` (subagent link plan/apply/adopt/prune),
  `internal/engine/status.go` (doctor), and `homonto.toml` (declare the loose
  subagents / enable via framework).
- New tests for subagent parsing/expansion, single-file materialization, and
  subagent projection into both tools.
- Reuses `internal/link` (already multi-root, variadic) and `internal/state`
  (catalog version) unchanged.
- Advances the roadmap's "Immediate Next Work" item 2 (subagent projection),
  leaving the `onto` binary as the remaining release blocker.
