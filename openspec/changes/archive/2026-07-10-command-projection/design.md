## Context

The archived `catalog-foundation-skills` change built the reusable machinery for
builtin resource projection: an embedded `go:embed` catalog, version-gated
materialization to `.homonto/catalog/`, a variadic multi-root `internal/link`,
`state.CatalogVersion`, framework dependency expansion, and per-adapter
`catalogRoot` resolution. Commands (`[commands.X]`) are already parsed and
validated by `internal/config` but are projected nowhere. This change adds the
command projection path on top of that foundation. The key structural difference
from skills: **a skill is a directory** (`catalog/skills/<n>/…`) while **a command
is a single file** (`catalog/commands/<n>.md`). Real command content is deferred;
one placeholder fixture command proves the machinery.

## Goals / Non-Goals

**Goals**
- Project `[commands.X]` (builtin + local, scope-aware) into Claude
  (`~/.claude/commands/<n>.md`) and OpenCode (`~/.config/opencode/command/<n>.md`).
- Add `catalog/commands/` to the embedded catalog with single-file materialization.
- Expand framework-declared commands (`framework.toml [commands]`) like skills.
- Extend both adapters and `doctor`; reuse `internal/link` and `state` unchanged.

**Non-Goals**
- Real command content / framework-declared command sets (populated later).
- Namespaced commands (`/<ns>:<name>`); flat commands only.
- Subagent projection (change C), model routing, remote registry.

## Decisions

### D1: Catalog commands area + single-file materialization

`catalog/commands/<n>.md` files, added to the root `catalog` package's
`//go:embed` directive (`all:frameworks all:skills all:commands version.txt`).
`internal/catalog` gains a command-file materializer: unlike `Materialize` (which
`RemoveAll`s a dir and walks a sub-FS), a command materializes by reading the
single embedded file and writing `.homonto/catalog/commands/<n>.md` (0644),
version-gated on the same `state.CatalogVersion`. Likely a
`MaterializeCommands(dstRoot, names)` sibling, or a generalized materializer that
handles both kinds.

### D2: Framework `[commands]` table + expansion

`framework.toml` gains an optional `[commands]` table mapping command name →
`commands/<n>.md`. The catalog loader parses it into `Framework.Commands`
(alongside `Skills`), validating each path exists in the embedded FS. `Expand`
returns commands as well as skills (or a parallel command expansion), transitive
and deduped, with cycle/collision reuse from the skills path.

### D3: Command path mapping

A `commandpath.Dir(tool, scope, home, projectRoot)` analog to `skillpath.Dir`:
- claude user → `~/.claude/commands`, project → `<repo>/.claude/commands`
- opencode user → `~/.config/opencode/command`, project → `<repo>/.opencode/command`
(note OpenCode uses the singular `command/`). Either a new package or an extended
`skillpath` with a resource-kind parameter — design phase decides.

### D4: Adapter command linking

Each adapter gains a `commandsDir(scope)` and resolves command source by scheme
(`builtin:` → `.homonto/catalog/commands/<n>.md`, else `homonto/commands/<n>.md`).
Command links reuse `internal/link` (already variadic multi-root) with the
managed roots extended to include the commands roots (`homonto/commands`,
`.homonto/catalog/commands`). Plan/apply/prune/adopt mirror the skill paths;
state key `command.<n>` parallels `skill.<n>`.

### D5: Config expansion for commands

`Config.ExpandedCommandEntriesForTool(tool)` mirrors
`ExpandedSkillEntriesForTool`: explicit `[commands.X]` plus framework-expanded
commands, inheriting scope/targets, with collision (explicit vs framework) and
cycle errors surfaced from `internal/catalog`.

### D6: Engine materialization orchestration

`materializeCatalog` extends to also collect declared builtin command names and
materialize them (single-file) before the adapter loop, under the same version
gate; `CatalogVersion` still records only after all materialization (skills +
commands) succeeds.

### D7: Placeholder fixture command

One `catalog/commands/<placeholder>.md` (e.g. a trivial informational command),
declared in `homonto.toml` for dogfooding, so apply materializes + links it and
`status`/`doctor` verify it with no drift. Documented as a placeholder pending
real content.

## Risks / Trade-offs

- [Single-file vs directory materialization divergence] → keep the command path a
  thin sibling of the skill path; avoid duplicating link/state logic (reuse the
  variadic link package and the `command.<n>` state convention).
- [Managed-root sprawl: four roots now] → adapters compute `managedRoots()` to
  include only non-empty roots (the empty-root guard from change A) across skills
  and commands.
- [OpenCode `command/` (singular) vs `commands/`] → encode in `commandpath.Dir`
  as the single source of truth, like `skillpath` did for the tools' differing
  skill dirs.
- [No real content] → mitigated by the placeholder fixture; the capability spec
  marks content as later work so no doc claims commands are populated.
