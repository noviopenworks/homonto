# Comet Design Handoff

- Change: subagent-projection
- Phase: design
- Mode: full
- Context hash: aa15aa6db9e34e36aab70a333b166146899e902e999a7fc8e0e4f99e08171db1

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/subagent-projection/proposal.md

- Source: openspec/changes/subagent-projection/proposal.md
- Lines: 1-79
- SHA256: e106da7539f1b7de9465291298cff526588e018a8aca2e510bf786932885da3b

```md
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

```

## openspec/changes/subagent-projection/design.md

- Source: openspec/changes/subagent-projection/design.md
- Lines: 1-120
- SHA256: d71530accac59682dda0914755f6947de5d324f92db7ba1421da9694f94d7e2e

```md
## Context

Homonto projects skills and commands from a bundled `go:embed` catalog into
Claude Code and OpenCode through a proven foundation: version-gated
materialization to `.homonto/catalog/<kind>/`, scope-aware symlinking via
`internal/link` (multi-root, conflict-safe), per-resource scope with
relocation, adopt/prune, and `doctor` verification. `[subagents.X]` already
parses in `internal/config` and already participates in `validateModels` (any
tool a subagent targets must define all three `[models.<tool>.<level>]`
levels), but no adapter/engine/plan step projects it — subagents are the last
declared resource kind that is parsed and then ignored at apply.

The immediately preceding `command-projection` change established the exact
pattern this change follows. The deep technical design is refined further in
the Comet Design phase; this document fixes the high-level architecture.

## Goals / Non-Goals

**Goals:**

- Make `[subagents.X]` functional: materialize + symlink declared subagents into
  Claude Code (`agents/`) and OpenCode (`agent/`), scope-aware, with doctor
  verification and adopt/prune, reusing the skills/commands foundation.
- Expand framework-declared `[subagents]` tables transitively, deduplicated, with
  explicit-entry collision as a config error.
- Ship three real bundled subagents (`code-reviewer`, `codebase-explorer`, and a
  comet-framework subagent) so the machinery carries genuine content.
- Materialize verbatim — no frontmatter rewriting, no model injection.

**Non-Goals:**

- The `onto` binary (separate release-blocking work).
- Injecting resolved model routes into subagent frontmatter; per-subagent model
  overrides.
- Namespaced subagents (`<ns>:<name>`); flat `<name>` only.
- Remote/registry subagent sources.
- Changing skills/commands behavior or the existing model-route validation.

## Decisions

**D1 — Mirror the command pipeline, do not generalize it yet.** Add a parallel
`subagent.*` path rather than refactoring skills/commands/subagents into one
generic resource loop. Rationale: the command pattern is fresh, well-tested, and
low-risk to replicate; a premature generalization would touch the working
skills/commands paths. A later change may unify the three once all three exist.
Alternative (generic resource abstraction now) rejected as scope creep that
risks regressions in shipped behavior.

**D2 — Single-file verbatim materialization.** Reuse the command single-file
model: `catalog/subagents/<name>.md` → `.homonto/catalog/subagents/<name>.md`,
byte-for-byte, version-gated on the same catalog version. No `RemoveAll` needed
(single-file overwrite). Rationale: subagents are single Markdown files with
frontmatter, like commands; a symlink to verbatim content keeps edits live and
avoids per-tool file rewriting. Alternative (resolve model route into
frontmatter at apply) rejected: it breaks the symlink-clean model, forks
per-tool content, and edges into the per-subagent-model non-goal.

**D3 — Per-tool agent directory naming.** Claude Code uses `agents/` (plural);
OpenCode uses `agent/` (singular), consistent with OpenCode's singular
`command/`. Encode this in a path helper (`internal/subagentpath` or an
extension of `commandpath`) mapping `(tool, scope) → dir`, so the singular/plural
split lives in one place. The exact real-layout directories are confirmed by
fixtures in build (see Risks). User scope: `~/.claude/agents/`,
`~/.config/opencode/agent/`. Project scope: `<repo>/.claude/agents/`,
`<repo>/.opencode/agent/`.

**D4 — Framework `[subagents]` table.** Extend `framework.toml` parsing and
`ExpandSubagents` exactly like `[commands]`: inherit framework `scope`/`targets`,
transitive across dependencies, dedupe by name, explicit-entry collision is an
error. The `comet` framework's `framework.toml` gains a `[subagents]` entry
pointing at a real bundled subagent so expansion is exercised end-to-end.

**D5 — Real content, not a placeholder.** Unlike `command-projection` (one
placeholder), ship `code-reviewer` and `codebase-explorer` as loose builtin
subagents and one comet-framework subagent. `code-reviewer` and
`codebase-explorer` are declared standalone in `homonto.toml` for dogfood; the
comet subagent is exercised via `[frameworks.comet]` expansion. Each is authored
as a valid single-file agent definition (frontmatter + body) usable by the tools
it targets.

**D6 — State keys and reuse.** New `subagent.<name>` state keys, handled in
Plan/Apply/ObserveHashes identically to `command.<name>` (symlink hash
`Hash(dst + " -> " + src)`, adopt, orphan prune, scope-switch relocate).
`internal/link` and `internal/state` (catalog version) are reused unchanged;
`managedRoots()` gains the subagent catalog root only when set.

## Risks / Trade-offs

- **Exact tool agent file format/layout not yet fixture-confirmed** → Build
  starts by adding real-layout fixtures for Claude `agents/` and OpenCode
  `agent/` (mirroring the skills/commands fixtures) and asserts projection
  against them before wiring adapters; the path helper isolates any correction to
  one place.
- **Authoring three real subagents couples content to machinery** → Keep each
  subagent minimal but valid; the projection tests assert linking/no-drift, not
  subagent behavior, so content quality does not gate the machinery.
- **Parallel `subagent.*` code duplicates command logic** → Accepted per D1;
  duplication is localized and mirrors existing tested code. Unification is a
  deliberate later step.
- **Model validation already fires for subagent-targeted tools** → No change
  needed; verify existing `validateModels`/`EnabledModelTools` already counts
  subagents so enabling one without model routes fails clearly (add a test if a
  gap exists).

## Migration Plan

Additive only. New catalog tree, new state-key prefix, new adapter/doctor
branches; no changes to existing skill/command/MCP/settings behavior. Rollback is
removing the declarations and applying (prunes the links) or reverting the
change. Dogfood: declare the two loose subagents (and keep `[frameworks.comet]`),
`apply`, then confirm `status` → `No drift` and `doctor` reports both tools'
subagent links OK.

## Open Questions

- Confirm OpenCode's project-scope agent directory is `<repo>/.opencode/agent/`
  (singular) and Claude's is `<repo>/.claude/agents/` (plural) against real tool
  layout fixtures during build.
- Whether the comet-framework subagent should target both tools or Claude only
  for the first release (default: match how comet skills are targeted).

```

## openspec/changes/subagent-projection/tasks.md

- Source: openspec/changes/subagent-projection/tasks.md
- Lines: 1-62
- SHA256: bf13ff0d55f1eeec8ab0008188bb04d9601b508a23b779d79e614c718128a567

```md
## 1. Tool-layout fixtures and path mapping (confirm real layout first)

- [ ] 1.1 Add real-layout test fixtures for Claude `agents/` (plural) and OpenCode `agent/` (singular), user and project scope, mirroring the skills/commands fixtures
- [ ] 1.2 Add `subagentpath.Dir(tool, scope, home, projectRoot)` (claude `.claude/agents` user / `<repo>/.claude/agents` project; opencode `.config/opencode/agent` user / `<repo>/.opencode/agent` project) — extend `commandpath` or add a sibling package
- [ ] 1.3 Unit tests for all tool/scope combinations, asserting the singular/plural split

## 2. Catalog subagent content and embed

- [ ] 2.1 Author `catalog/subagents/code-reviewer.md` (framework-agnostic loose subagent, valid frontmatter + body)
- [ ] 2.2 Author `catalog/subagents/codebase-explorer.md` (read-only research subagent, valid frontmatter + body)
- [ ] 2.3 Author one comet-framework subagent under `catalog/subagents/<name>.md`
- [ ] 2.4 Extend the root `catalog` package `//go:embed` directive to include `all:subagents`
- [ ] 2.5 Verify the embed compiles and all three subagents are present in the embedded FS

## 3. Catalog subagent loading, expansion, materialization

- [ ] 3.1 Parse an optional `[subagents]` table into `Framework.Subagents` (name → `subagents/<n>.md`); validate each path exists in the embedded FS
- [ ] 3.2 Index subagents and add a subagent-path lookup (`SubagentPath(name)`)
- [ ] 3.3 Add `ExpandSubagents` (transitive, deduped), mirroring `ExpandCommands`
- [ ] 3.4 Add single-file **verbatim** materialization to `.homonto/catalog/subagents/<n>.md`, version-gated (assert byte-for-byte equal to source)
- [ ] 3.5 Add the comet framework's `[subagents]` entry to `catalog/frameworks/comet/framework.toml`
- [ ] 3.6 Unit tests: subagent table parse, expansion/dedup, single-file materialize, missing-file re-materialize, no-model-injection (content equals source)

## 4. Config subagent expansion

- [ ] 4.1 Add `Config.ExpandedSubagentEntriesForTool(tool)` (explicit `[subagents.X]` + framework-expanded subagents, scope/targets inheritance)
- [ ] 4.2 Collision detection (explicit vs framework subagent name) and cycle propagation
- [ ] 4.3 Verify `EnabledModelTools`/`validateModels` already counts subagent-targeted tools; add a test asserting a subagent enabling a tool without model routes fails clearly
- [ ] 4.4 Config tests for subagent expansion, inheritance, collision, target filtering

## 5. Engine materialization orchestration

- [ ] 5.1 Extend catalog materialization to collect declared builtin subagent names and materialize them (single-file) before adapters, under the same version gate
- [ ] 5.2 Ensure `CatalogVersion` is recorded only after skills + commands + subagents materialization succeeds
- [ ] 5.3 Add `WithSubagentCatalogRoot` wiring for both adapters
- [ ] 5.4 Engine tests: first-apply subagent materialization, version-gated skip, missing-file refresh

## 6. Adapter subagent projection

- [ ] 6.1 Claude adapter: `subagentsDir(scope)`, `inactiveSubagentsDir`, `subagentSource(entry)`, `subagentLinks`, plan/apply/adopt/prune for `subagent.<n>` links via variadic managed roots
- [ ] 6.2 OpenCode adapter: same, using `subagentpath` (singular `agent/`)
- [ ] 6.3 Extend `managedRoots()` to include the subagent catalog root (non-empty guard)
- [ ] 6.4 `ObserveHashes`: handle `subagent.<n>` as a symlink hash, mirroring `command.<n>`
- [ ] 6.5 Adapter tests (both tools): builtin subagent link create, idempotent re-apply, conflict-not-clobbered, de-declared prune, scope-switch relocate, adopt pre-existing link, state `subagent.<n>` recorded

## 7. Doctor

- [ ] 7.1 Extend `doctor` to verify subagent links and materialized subagent files for both tools
- [ ] 7.2 Doctor test for a linked builtin subagent (both tools)

## 8. Dogfood

- [ ] 8.1 Declare `code-reviewer` and `codebase-explorer` in `homonto.toml` (builtin, scope project); keep `[frameworks.comet]` for the framework subagent
- [ ] 8.2 Run `homonto apply --yes`; verify materialize + link of all three subagents into targeted tools
- [ ] 8.3 Run `homonto status` (No drift) and `homonto doctor` (subagent links ok for both tools)

## 9. Regression and docs

- [ ] 9.1 Full regression: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l .`
- [ ] 9.2 Stale-doc grep: no doc claims subagent projection is unimplemented once shipped; update README "Known limitations" and `docs/guides/using-homonto.md`
- [ ] 9.3 Update `docs/roadmap.md` v1.1 status (subagent projection landed with real content) and the "Immediate Next Work" section (item 2 done; onto binary remains)
- [ ] 9.4 Commit all changes

```

## openspec/changes/subagent-projection/specs/config-model/spec.md

- Source: openspec/changes/subagent-projection/specs/config-model/spec.md
- Lines: 1-41
- SHA256: 50ec0ab7555ce03889c19ab3dcbf0a19e3e62f54b22a3475d2bdcf7f814acb1f

```md
## MODIFIED Requirements

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory containing `homonto.toml`; generated state, cache, and the materialized builtin catalog SHALL live under `.homonto/` only. Current adapters resolve local-source skills (`source = "local:<name>"`) from `homonto/skills/<name>`, local-source commands from `homonto/commands/<name>.md`, and local-source subagents from `homonto/subagents/<name>.md`. Builtin-source skills resolve from the materialized `.homonto/catalog/skills/<name>/`, builtin-source commands from `.homonto/catalog/commands/<name>.md`, and builtin-source subagents from `.homonto/catalog/subagents/<name>.md`. Local framework content resolution beyond these resource kinds is part of future framework/catalog projection work and MUST NOT be claimed as installed behavior yet.

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

#### Scenario: Local subagent resolves from homonto/subagents

- **GIVEN** a config with `[subagents.mine] source = "local:mine"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `homonto/subagents/mine.md`

#### Scenario: Builtin subagent resolves from materialized catalog

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `.homonto/catalog/subagents/code-reviewer.md`

```

## openspec/changes/subagent-projection/specs/framework-expansion/spec.md

- Source: openspec/changes/subagent-projection/specs/framework-expansion/spec.md
- Lines: 1-23
- SHA256: 18eadf0a37f2e9756664dca2da23fe97e5df8bc0b54d4d7253e2c5ecbf9012ad

```md
## MODIFIED Requirements

### Requirement: Framework metadata format

Each framework in the catalog SHALL have a `framework.toml` metadata file declaring `name`, `version`, `description`, optional `[dependencies] frameworks` list, and resource lists by kind (`[skills]`, `[commands]`, and `[subagents]`). Each resource entry SHALL map a resource name to a catalog-relative path (`skills/<name>` for a skill directory, `commands/<name>.md` for a command file, `subagents/<name>.md` for a subagent file).

#### Scenario: Parse framework metadata

- **GIVEN** a framework `catalog/frameworks/comet/framework.toml` with name, version, dependencies, and a skills table
- **WHEN** Homonto loads the framework
- **THEN** it exposes the framework name, version, dependency names, and a map of skill names to catalog paths

#### Scenario: Parse framework command table

- **GIVEN** a framework `framework.toml` declaring a `[commands]` table mapping `demo-cmd = "commands/demo-cmd.md"`
- **WHEN** Homonto loads the framework
- **THEN** it exposes a map of command names to catalog command-file paths alongside the skills map

#### Scenario: Parse framework subagent table

- **GIVEN** a framework `framework.toml` declaring a `[subagents]` table mapping `demo-agent = "subagents/demo-agent.md"`
- **WHEN** Homonto loads the framework
- **THEN** it exposes a map of subagent names to catalog subagent-file paths alongside the skills and commands maps

```

## openspec/changes/subagent-projection/specs/subagent-projection/spec.md

- Source: openspec/changes/subagent-projection/specs/subagent-projection/spec.md
- Lines: 1-123
- SHA256: 2d8dfbb426a6eedf9393d66ad04138e4b7c15a820829673e84c296b01fc95670

```md
## ADDED Requirements

### Requirement: Builtin and local subagent source resolution

A subagent resource SHALL resolve its content by source scheme: `[subagents.<name>] source = "builtin:<name>"` resolves from the embedded catalog at `catalog/subagents/<name>.md` (materialized to `.homonto/catalog/subagents/<name>.md` on apply), and `source = "local:<name>"` resolves from `homonto/subagents/<name>.md`. Subagents are single Markdown files, not directories. Every subagent resource SHALL declare a `scope` (`user` or `project`) exactly as skills and commands do.

#### Scenario: Builtin subagent resolves from materialized catalog

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"` and `scope = "user"`
- **WHEN** apply runs
- **THEN** `catalog/subagents/code-reviewer.md` is materialized to `.homonto/catalog/subagents/code-reviewer.md` and the subagent link targets that file

#### Scenario: Local subagent resolves from homonto/subagents

- **GIVEN** a config with `[subagents.mine] source = "local:mine"` and `scope = "project"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `homonto/subagents/mine.md`

### Requirement: Single-file verbatim subagent materialization

Homonto SHALL materialize builtin subagent content as single files from the embedded catalog to `.homonto/catalog/subagents/<name>.md` before creating subagent symlinks, version-gated on the same catalog version tracked in state as skills and commands. The materialized file SHALL be byte-for-byte identical to the embedded catalog source: Homonto SHALL NOT rewrite the subagent's frontmatter, and SHALL NOT inject a resolved model route into the projected file. Re-materialization SHALL occur only when the catalog version changes or the target file is missing, and the catalog version SHALL be recorded only after a successful materialization.

#### Scenario: First subagent materialization

- **GIVEN** no `.homonto/catalog/subagents/code-reviewer.md` exists
- **WHEN** apply runs with a config declaring a builtin subagent `code-reviewer`
- **THEN** `.homonto/catalog/subagents/code-reviewer.md` is written byte-for-byte from the embedded catalog

#### Scenario: Version-gated subagent skip

- **GIVEN** `.homonto/catalog/subagents/code-reviewer.md` exists and state records the current catalog version
- **WHEN** apply runs again with the same binary
- **THEN** the subagent is not re-materialized and the link is a no-op

#### Scenario: Model route is not injected

- **GIVEN** a config whose `[models.<tool>.<level>]` routes are defined and a builtin subagent is declared
- **WHEN** apply materializes and links the subagent
- **THEN** the projected file's content equals the catalog source and contains no Homonto-injected model value

### Requirement: Subagent projection into tool agent directories

Owned subagents SHALL be linked (not copied) into each tool's agent directory at the location chosen by the resource's `scope`: Claude Code at `~/.claude/agents/<name>.md` (user) or `<repo>/.claude/agents/<name>.md` (project), and OpenCode at `~/.config/opencode/agent/<name>.md` (user) or `<repo>/.opencode/agent/<name>.md` (project). Claude Code uses the plural `agents/` directory and OpenCode uses the singular `agent/` directory. Pending link work SHALL appear as plan changes (create / update / no-op). `apply` SHALL record each applied subagent link in state and SHALL prune a de-declared subagent's link only when it is a symlink pointing into a homonto-managed root (`homonto/subagents/` or `.homonto/catalog/subagents/`); a real file or foreign link SHALL be reported as a conflict and never clobbered. A per-resource `scope` switch SHALL appear as a relocation that removes the old-scope link as it creates the new one.

#### Scenario: Builtin subagent links into both tools

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"` targeting claude and opencode
- **WHEN** apply runs
- **THEN** `~/.claude/agents/code-reviewer.md` and `~/.config/opencode/agent/code-reviewer.md` are symlinks into `.homonto/catalog/subagents/code-reviewer.md`

#### Scenario: Idempotent subagent link

- **WHEN** a subagent link already points at its materialized target
- **THEN** plan reports no change and a second apply is a no-op

#### Scenario: Conflict is reported, not clobbered

- **GIVEN** a real file already exists at the subagent's link destination
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: De-declared subagent pruned only when it is our link

- **GIVEN** a subagent removed from `homonto.toml` whose link is a symlink into a homonto-managed root
- **WHEN** apply processes the delete
- **THEN** the link is removed; a real file or foreign link at that path is instead reported as a conflict and left untouched

#### Scenario: Scope switch relocates the link

- **GIVEN** a subagent whose `scope` changes from `user` to `project` (or the reverse)
- **WHEN** apply runs
- **THEN** the plan shows a relocation and apply removes the old-scope link while creating the new-scope link

### Requirement: Subagent adoption of pre-existing matching links

A correct-but-unrecorded subagent link — one already on disk pointing at its materialized or local content but absent from (or stale in) state — SHALL be adopted into state without rewriting the on-disk link, exactly as skill and command links are adopted, so a lost `state.json` can be rebuilt without a spurious change.

#### Scenario: Adopt an already-correct subagent link

- **GIVEN** a subagent link on disk that already points at its content but is not recorded in state
- **WHEN** apply runs
- **THEN** the link is left untouched and its record is added to state as an adoption (no create/update)

### Requirement: Framework subagent expansion

A `framework.toml` `[subagents]` table SHALL expand through `[frameworks.<name>] source = "builtin:<framework>"` into effective subagent resources with `source = "builtin:<subagent-name>"`, each inheriting the framework declaration's `scope` and `targets`, transitively across dependency frameworks and deduplicated by name, exactly as skills and commands expand. A subagent name colliding with an explicit `[subagents.X]` entry SHALL be a config error.

#### Scenario: Framework expands its subagents

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` where the comet framework declares a subagent in its `[subagents]` table
- **WHEN** the config is loaded
- **THEN** the effective subagent set includes that subagent as a builtin-source subagent inheriting the framework's scope and targets

### Requirement: Subagent link doctor verification

`doctor` SHALL verify each recorded subagent link: a builtin subagent's materialized target under `.homonto/catalog/subagents/` SHALL exist, and the tool-side symlink SHALL be present and point at the expected source; a missing materialized file or broken link SHALL be reported like a broken skill or command link, for both Claude Code and OpenCode.

#### Scenario: Doctor reports a linked subagent

- **GIVEN** a builtin subagent materialized and linked into a tool
- **WHEN** `doctor` runs
- **THEN** it reports the subagent link as present and correct for that tool

### Requirement: Bundled real subagents

The first release of this capability SHALL ship real subagent content in `catalog/subagents/`, not only a placeholder: at least `code-reviewer` and `codebase-explorer` as framework-agnostic loose builtin subagents, plus one subagent declared in the `comet` framework's `[subagents]` table so framework-declared subagent expansion is exercised with real content. Each bundled subagent SHALL be a valid single-file agent definition for the tools it targets.

#### Scenario: Loose builtin subagents are projectable

- **GIVEN** the bundled catalog containing `code-reviewer` and `codebase-explorer`
- **WHEN** each is declared as `[subagents.X] source = "builtin:X"` and applied
- **THEN** it materializes and links into its targeted tools with no drift on a second status

#### Scenario: Comet framework subagent expands and projects

- **GIVEN** `[frameworks.comet]` enabled and the comet framework declaring a subagent in `[subagents]`
- **WHEN** apply runs
- **THEN** that subagent is materialized and linked into the framework's targeted tools alongside comet's skills

#### Scenario: Shared minimal frontmatter is valid for both tools

- **GIVEN** a bundled subagent targeting both Claude Code and OpenCode whose single file carries only minimal shared frontmatter (`name`, `description`, `mode: subagent`) and omits `model` and `tools`
- **WHEN** the same materialized file is linked into both tools
- **THEN** it is a valid agent definition for each tool, and each tool applies its own default model and tool set (no Homonto-injected model or tools)

```
