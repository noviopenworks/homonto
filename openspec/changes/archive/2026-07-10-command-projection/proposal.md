## Why

Homonto's config model parses `[commands.X]` resources with `source =
"builtin:<name>"` / `"local:<name>"`, and the archived `catalog-foundation-skills`
change built the reusable foundation (embedded catalog, materialization,
version-gated caching, managed-root symlinking) — but only skills are projected.
Commands are still parsed and then ignored at apply. This change adds the command
projection machinery so `[commands.X]` and framework-declared commands install
into Claude Code and OpenCode, reusing that foundation. Real command content is
deliberately deferred: the catalog's commands (and broader skills/frameworks) are
populated in a future change; this change ships only the machinery plus one
placeholder fixture command to prove and dogfood it end-to-end.

## What Changes

- Add builtin/local **command** projection: a `[commands.<name>]` resource
  (`source = "builtin:<name>"` or `"local:<name>"`, with a required `scope`)
  links a single `.md` file into Claude (`~/.claude/commands/<name>.md`) and
  OpenCode (`~/.config/opencode/command/<name>.md`), scope-aware like skills.
- Add a `catalog/commands/<name>.md` area to the embedded catalog and
  **single-file** materialization to `.homonto/catalog/commands/<name>.md`
  (skills materialize as directories; commands are single files).
- Extend `framework.toml` with an optional `[commands]` table and expand
  framework-declared commands through `[frameworks.X]`, transitively, mirroring
  skill expansion.
- Extend both adapters and `doctor` to plan/apply/prune/verify command links,
  reusing the managed-root and version-gated materialization foundation.
- Add one placeholder fixture command to `catalog/commands/` so the machinery is
  materialized, linked, and dogfooded (real content lands later).
- Commands are **flat** only in this change (`commands/<name>.md` → `/<name>`);
  namespaced commands (`/<ns>:<name>`) and real bundled content are non-goals.

## Capabilities

### New Capabilities

- `command-projection`: builtin/local command source resolution, single-file
  materialization from the embedded catalog, projection into Claude Code and
  OpenCode command directories with conflict-safe managed-root linking and
  pruning, framework `[commands]` expansion, and doctor verification of command
  links.

### Modified Capabilities

- `config-model`: `[commands.X]` gains projection behavior (materialize + link),
  not just parse/validate; the "Local provider content root" requirement's claim
  that command resolution is future work no longer holds for commands.
- `framework-expansion`: the framework metadata format's `[commands]` table
  becomes an expanded resource kind (previously reserved as "later").

## Impact

- New `catalog/commands/` tree (one placeholder command) embedded via `go:embed`.
- New `internal/commandpath` (or extended `skillpath`) mapping `(tool, scope)` to
  a command directory.
- Modified `internal/catalog` (single-file materialization for commands),
  `internal/config` (`ExpandedCommandEntriesForTool` + framework `[commands]`
  expansion), `internal/engine` (materialize commands alongside skills),
  `internal/adapter/{claude,opencode}` (command link plan/apply/prune),
  `internal/engine/status.go` (doctor), and `homonto.toml` (declare the fixture).
- New tests for command parsing/expansion, single-file materialization, and
  command projection into both tools.
- Reuses `internal/link` (already multi-root, variadic) and `internal/state`
  (catalog version) unchanged.
