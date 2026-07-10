# Comet Design Handoff

- Change: command-projection
- Phase: design
- Mode: compact
- Context hash: 279b83b2bd0db0b750fc3e46a2152c2aeed5591ff16115bb251483a4e867c682

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/command-projection/proposal.md

- Source: openspec/changes/command-projection/proposal.md
- Lines: 1-64
- SHA256: b7149ee2c375db4823d0030a3e309c284d392aad924e2ca712a7ea09b7702ff0

```md
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

```

## openspec/changes/command-projection/design.md

- Source: openspec/changes/command-projection/design.md
- Lines: 1-99
- SHA256: 825756ea9e79fd04c2f5ad6c7ff32bf1aa085e4d0df6099ba5b2ba62c2a67ad5

[TRUNCATED]

```md
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

```

Full source: openspec/changes/command-projection/design.md

## openspec/changes/command-projection/tasks.md

- Source: openspec/changes/command-projection/tasks.md
- Lines: 1-55
- SHA256: ef7cf22b27a2bee4e8fd8f0e89b19028831ae43b049f7d88a1bc7fb81368001d

```md
## 1. Catalog commands content and embed

- [ ] 1.1 Add `catalog/commands/<placeholder>.md` (one flat placeholder command with frontmatter)
- [ ] 1.2 Extend the root `catalog` package `//go:embed` directive to include `all:commands`
- [ ] 1.3 Verify the embed compiles and the placeholder command is present in the embedded FS

## 2. Catalog command loading, expansion, materialization

- [ ] 2.1 Parse an optional `[commands]` table into `Framework.Commands` (name → `commands/<n>.md`); validate each path exists in the embedded FS
- [ ] 2.2 Index commands and add a command-path lookup (`CommandPath(name)`)
- [ ] 2.3 Expand framework commands (transitive, deduped) — extend `Expand` or add `ExpandCommands`
- [ ] 2.4 Add single-file command materialization to `.homonto/catalog/commands/<n>.md`, version-gated
- [ ] 2.5 Unit tests: command table parse, command expansion/dedup, single-file materialize, missing-file re-materialize

## 3. Command path mapping

- [ ] 3.1 Add `commandpath.Dir(tool, scope, home, projectRoot)` (claude `.claude/commands`, opencode `.config/opencode/command` user / `.opencode/command` project)
- [ ] 3.2 Unit tests for all tool/scope combinations

## 4. Config command expansion

- [ ] 4.1 Add `Config.ExpandedCommandEntriesForTool(tool)` (explicit `[commands.X]` + framework-expanded commands, scope/targets inheritance)
- [ ] 4.2 Collision detection (explicit vs framework command name) and cycle propagation
- [ ] 4.3 Config tests for command expansion, inheritance, collision, target filtering

## 5. Engine materialization orchestration

- [ ] 5.1 Extend `materializeCatalog` to collect declared builtin command names and materialize them (single-file) before adapters, under the same version gate
- [ ] 5.2 Ensure `CatalogVersion` is recorded only after skills + commands materialization succeeds
- [ ] 5.3 Engine tests: first-apply command materialization, version-gated skip, missing-file refresh

## 6. Adapter command projection

- [ ] 6.1 Claude adapter: `commandsDir(scope)`, `commandSource(entry)`, plan/apply/prune/adopt for `command.<n>` links via variadic managed roots
- [ ] 6.2 OpenCode adapter: same, using `commandpath` (singular `command/`)
- [ ] 6.3 Extend `managedRoots()` to include the commands roots (non-empty guard)
- [ ] 6.4 Adapter tests (both tools): builtin command link create, idempotent re-apply, conflict-not-clobbered, de-declared prune, state `command.<n>` recorded

## 7. Doctor

- [ ] 7.1 Extend `doctor` to verify command links and materialized command files
- [ ] 7.2 Doctor test for a linked builtin command

## 8. Dogfood

- [ ] 8.1 Declare the placeholder command in `homonto.toml` (builtin, scope project)
- [ ] 8.2 Run `homonto apply --yes`; verify materialize + link into both tools
- [ ] 8.3 Run `homonto status` (No drift) and `homonto doctor` (command link ok)

## 9. Regression and docs

- [ ] 9.1 Full regression: `go test ./... -count=1`, `go vet ./...`, `go build ./...`
- [ ] 9.2 Stale-doc grep: no doc claims command projection is unimplemented for skills-and-commands once shipped
- [ ] 9.3 Update `docs/roadmap.md` v1.1 status (command projection machinery landed; content deferred)
- [ ] 9.4 Commit all changes

```

## openspec/changes/command-projection/specs/command-projection/spec.md

- Source: openspec/changes/command-projection/specs/command-projection/spec.md
- Lines: 1-109
- SHA256: 24fcc481ac8d7485ed18fb983458f51c9d603b6fe11d0b49c004887dba70baf4

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: Builtin and local command source resolution

A command resource SHALL resolve its content by source scheme: `[commands.<name>] source = "builtin:<name>"` resolves from the embedded catalog at `catalog/commands/<name>.md` (materialized to `.homonto/catalog/commands/<name>.md` on apply), and `source = "local:<name>"` resolves from `homonto/commands/<name>.md`. Commands are single Markdown files, not directories. Every command resource SHALL declare a `scope` (`user` or `project`) exactly as skills do.

#### Scenario: Builtin command resolves from materialized catalog

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"` and `scope = "user"`
- **WHEN** apply runs
- **THEN** `catalog/commands/demo.md` is materialized to `.homonto/catalog/commands/demo.md` and the command link targets that file

#### Scenario: Local command resolves from homonto/commands

- **GIVEN** a config with `[commands.mine] source = "local:mine"` and `scope = "project"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `homonto/commands/mine.md`

### Requirement: Single-file command materialization

Homonto SHALL materialize builtin command content as single files from the
embedded catalog to `.homonto/catalog/commands/<name>.md` before creating command
symlinks, version-gated on the same catalog version tracked in state as skills.
Re-materialization SHALL occur only when the catalog version changes or the target
file is missing, and the catalog version SHALL be recorded only after a
successful materialization.

#### Scenario: First command materialization

- **GIVEN** no `.homonto/catalog/commands/demo.md` exists
- **WHEN** apply runs with a config declaring a builtin command `demo`
- **THEN** `.homonto/catalog/commands/demo.md` is written from the embedded catalog

#### Scenario: Version-gated command skip

- **GIVEN** `.homonto/catalog/commands/demo.md` exists and state records the current catalog version
- **WHEN** apply runs again with the same binary
- **THEN** the command is not re-materialized and the link is a no-op

### Requirement: Command projection into tool command directories

Owned commands SHALL be linked (not copied) into each tool's command directory at
the location chosen by the resource's `scope`: Claude at
`~/.claude/commands/<name>.md` (user) or `<repo>/.claude/commands/<name>.md`
(project), and OpenCode at `~/.config/opencode/command/<name>.md` (user) or
`<repo>/.opencode/command/<name>.md` (project). Pending link work SHALL appear as
plan changes (create / update / no-op). `apply` SHALL record each applied command
link in state and SHALL prune a de-declared command's link only when it is a
symlink pointing into a homonto-managed root (`homonto/commands/` or
`.homonto/catalog/commands/`); a real file or foreign link SHALL be reported as a
conflict and never clobbered.

#### Scenario: Builtin command links into both tools

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"` targeting claude and opencode
- **WHEN** apply runs
- **THEN** `~/.claude/commands/demo.md` and `~/.config/opencode/command/demo.md` are symlinks into `.homonto/catalog/commands/demo.md`

#### Scenario: Idempotent command link

- **WHEN** a command link already points at its materialized target
- **THEN** plan reports no change and a second apply is a no-op

#### Scenario: Conflict is reported, not clobbered

- **GIVEN** a real file already exists at the command's link destination
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: De-declared command pruned only when it is our link

- **GIVEN** a command removed from `homonto.toml` whose link is a symlink into a homonto-managed root
- **WHEN** apply processes the delete
- **THEN** the link is removed; a real file or foreign link at that path is instead reported as a conflict and left untouched

### Requirement: Framework command expansion

A `framework.toml` `[commands]` table SHALL expand through `[frameworks.<name>] source = "builtin:<framework>"` into effective command resources with `source = "builtin:<command-name>"`, each inheriting the framework declaration's `scope` and `targets`, transitively across dependency frameworks and deduplicated by name, exactly as skills expand. A command name colliding with an explicit `[commands.X]` entry SHALL be a config error.

#### Scenario: Framework expands its commands


```

Full source: openspec/changes/command-projection/specs/command-projection/spec.md

## openspec/changes/command-projection/specs/config-model/spec.md

- Source: openspec/changes/command-projection/specs/config-model/spec.md
- Lines: 1-29
- SHA256: e2e05d60328e4efd113606df8da4c18172fb54977cc0f8b54de8c6632507dab3

```md
## MODIFIED Requirements

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory containing `homonto.toml`; generated state, cache, and the materialized builtin catalog SHALL live under `.homonto/` only. Current adapters resolve local-source skills (`source = "local:<name>"`) from `homonto/skills/<name>` and local-source commands from `homonto/commands/<name>.md`. Builtin-source skills resolve from the materialized `.homonto/catalog/skills/<name>/` and builtin-source commands from the materialized `.homonto/catalog/commands/<name>.md`. Local subagent and framework content resolution is part of future framework/catalog projection work and MUST NOT be claimed as installed behavior yet.

#### Scenario: Local skill resolves from homonto/

- **GIVEN** a config with `[skills.my-skill] source = "local:my-skill"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `homonto/skills/my-skill/`

#### Scenario: Builtin skill resolves from materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `.homonto/catalog/skills/brainstorming/`

#### Scenario: Local command resolves from homonto/commands

- **GIVEN** a config with `[commands.mine] source = "local:mine"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `homonto/commands/mine.md`

#### Scenario: Builtin command resolves from materialized catalog

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `.homonto/catalog/commands/demo.md`

```

## openspec/changes/command-projection/specs/framework-expansion/spec.md

- Source: openspec/changes/command-projection/specs/framework-expansion/spec.md
- Lines: 1-17
- SHA256: 55aaa2d932bc34b7d84a101397914db02435ff3c7456ffb6b2f5f9f48454fc01

```md
## MODIFIED Requirements

### Requirement: Framework metadata format

Each framework in the catalog SHALL have a `framework.toml` metadata file declaring `name`, `version`, `description`, optional `[dependencies] frameworks` list, and resource lists by kind (`[skills]` and `[commands]`, and later `[subagents]`). Each resource entry SHALL map a resource name to a catalog-relative path (`skills/<name>` for a skill directory, `commands/<name>.md` for a command file).

#### Scenario: Parse framework metadata

- **GIVEN** a framework `catalog/frameworks/comet/framework.toml` with name, version, dependencies, and a skills table
- **WHEN** Homonto loads the framework
- **THEN** it exposes the framework name, version, dependency names, and a map of skill names to catalog paths

#### Scenario: Parse framework command table

- **GIVEN** a framework `framework.toml` declaring a `[commands]` table mapping `demo-cmd = "commands/demo-cmd.md"`
- **WHEN** Homonto loads the framework
- **THEN** it exposes a map of command names to catalog command-file paths alongside the skills map

```
