# homonto — Post-v1 Roadmap

**Date:** 2026-07-10
**Status:** Product roadmap. Release-readiness tasks live in
[`road-to-release.md`](road-to-release.md).

## Summary

`homonto` v1 remains focused on the safe core: one declarative
`homonto.toml`, a plan/confirm/apply pipeline, reference-only secrets,
surgical writes, and Claude Code/OpenCode adapters. The core is implemented and
testable (168 tests across 16 packages locally on 2026-07-09). The explicit
per-resource config model — `[frameworks.X]`, `[skills.X]`, `[commands.X]`,
`[subagents.X]`, `[models.<tool>.<level>]` with required `source` + `scope` and
local provider content under `homonto/` — has landed. The first public release
gate has been **reopened** for a dual-binary `homonto` + `onto` product; see
[`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`](superpowers/specs/2026-07-09-dual-binary-release-design.md),
which supersedes the prior "release-ready pending the maintainer's tag" verdict.
What remains beyond that is either consciously accepted as a documented
limitation or belongs to the post-v1 roadmap below.

Post-v1 expands Homonto from a config projector into a manager for the AI
coding-tool ecosystem around those configs: framework/catalog projection,
workflow operation through `onto`, richer plugin configuration, Claude/OpenCode
TUI-related settings, and full lifecycle management for agents.

## Immediate Next Work

One item stands between the current `main` and the reopened dual-binary
release gate, scoped in
[`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`](superpowers/specs/2026-07-09-dual-binary-release-design.md);
it is not optional for `v0.1.0-rc.1`. Its foundation has landed on the
`onto-binary-foundation` change (not yet merged to `main`); the rest is
still open, as detailed below.

1. **`onto` binary (release-blocking) — foundation, `onto init`, and change
   skeleton creation landed, work remains.** The binary foundation has
   landed: a second `package main` at `cmd/onto` builds an `onto` binary
   alongside `homonto`; `internal/ontostate` models `onto-state.yaml` (parse,
   validate, derive phase; phase set `open|design|build|verify|close`); and
   `onto status` is the read-only, config-independent command — it globs
   `docs/changes/*/onto-state.yaml` and prints each active change's derived
   phase, never reading `homonto.toml` and never writing a file. `onto init`
   has also landed: it idempotently scaffolds the `docs/{changes,specs,adr,
   guides}` layout, gated behind the Homonto framework install — it writes
   nothing and directs the user to initialize/apply Homonto first if
   `[frameworks.onto]` is not installed, and never overwrites user files on
   repeat runs. Change skeleton creation has also landed (#3a): `onto
   new <change>` creates a gated, no-clobber change skeleton
   (`onto-state.yaml`, `proposal.md`, `tasks.md`) under
   `docs/changes/<change>/`, refusing to run without the Homonto framework
   installed and refusing to overwrite an existing change directory; the
   `internal/ontostate` model gained a writer plus phase-aware skeleton
   validation, and `onto status` now reports each change's skeleton validity
   alongside its derived phase. Gated phase transitions have also landed
   (#3b): `onto advance <change>` moves a change through the fixed phase
   order `open → design → build → verify → close` only when the *current*
   phase's required deliverables are complete (and, to leave `build`, all
   tasks in `tasks.md` are checked); a normal advance on a dirty worktree
   prints a warning and proceeds, while the release-critical `verify →
   close` transition is blocked outright on a dirty or indeterminate
   worktree, with no phase write on any refusal. Archive and close rules
   have also landed (#3c): `onto close <change>` archives a close-phase
   change by moving `docs/changes/<name>/` → `docs/changes/archive/<YYYY-MM-DD>-<name>/`
   and setting `archived: true`, gated on every dependency being resolved
   (an `internal/ontostate.DepsResolved` helper treats a dep as resolved
   only once an archived `docs/changes/archive/*-<dep>` directory exists)
   and on a clean worktree, refusing (no move, no state write) on a
   non-close phase, an unresolved dep, a dirty worktree, or a pre-existing
   archive target. This completes the onto workflow engine — the `onto`
   binary can now create (`onto new`), advance (`onto advance`), and close
   (`onto close`) a change, enforcing the full phase-transition, archive,
   and dependency invariants. Workflow/project health checks have also
   landed (#4): `onto doctor` is a read-only, config-independent, ungated
   diagnostic that reports an onto workspace's health — docs layout
   (`docs/{changes,specs,adr,guides}`), `onto-state.yaml` validity,
   phase-derivation-matches-artifacts, dependency and archived-flag
   consistency, and archive-layout validity — printing each problem as a
   finding and exiting non-zero when any exist (so CI and smoke tests can
   gate on it), or `healthy` and exit 0 otherwise. It writes nothing and
   imports none of homonto's projection pipeline, as the peer to `homonto
   doctor`'s installation/projection health. The `onto` binary now exposes
   advance / close / doctor / init / new / status / version. Dual-binary
   release packaging has also landed (#5): the release pipeline now ships
   **both** binaries. A shared, locally-runnable `scripts/build-release.sh`
   cross-compiles `homonto` and `onto` for all six OS/arch targets as
   separate per-binary archives (12 in total), each version-stamped via its
   own package (`internal/cli`, `internal/ontocli`), under one shared
   `SHA256SUMS`; `release.yml` calls the script, and `ci.yml` gained an
   `onto` version-stamp smoke. **This completes the onto binary work (#1–#5)
   and the last release-gate packaging task.** What remains before the
   `v0.1.0-rc.1` tag is the maintainer's action to cut it (plus any
   remaining release-gate coverage the maintainer wants — e.g. the Docker
   smoke tests enumerated in the dual-binary design's release gate).

Skills, command, and subagent projection have all landed on `main` (see v1.1
below), and the dual-binary `homonto` + `onto` product — foundation,
framework install, workflow engine (create/advance/close), doctor, and
release packaging — is complete on `main`. The maintainer-owned
`v0.1.0-rc.1` tag is the remaining gate.

## Roadmap Strategy

Use a layered roadmap instead of expanding v1. The current v1 implementation
plan writes into real user tool configuration, so the first milestone must prove
safety, idempotency, drift detection, and surgical merge behavior before adding
broader product surface area.

Phases:

1. **v1 Core** — existing implementation plan, with safety/idempotency fixes.
2. **v1.1 Onto Framework And Catalog Projection** — curated bundled frameworks
   and loose skills, commands, and subagents projected into supported tools.
3. **v1.2 Plugin Configuration** — plugin install/enable declarations plus
   plugin-specific configuration.
4. **v1.3 Tool TUI Configuration** — Claude/OpenCode TUI-related plugins,
   themes, display settings, and keybindings where supported.
5. **v2 Agent Lifecycle** — sources, versions, compatibility, updates,
   migrations, and conflict handling for agents.

## v1 Core

The current v1 implementation remains the foundation:

- `homonto.toml` is the source of truth.
- `homonto plan` shows safe diffs without resolving secret values.
- `homonto apply` confirms, resolves secrets, writes atomically, and updates
  local state last.
- `homonto init`, `import`, `status`, and `doctor` provide adoption and health
  workflows.
- Claude Code and OpenCode adapters project MCPs, owned content, plugins, and
  settings into each tool.

Implemented and verified since the original v1 review:

- Claude MCPs project with the real schema (`command` string plus `args`).
- Import preserves Claude `command` plus `args`.
- Plans redact missing-state or unknown-provenance old values.
- State stores unresolved desired values plus non-secret applied hashes.
- State-recorded pruning exists for MCPs, settings, plugins, and skills.
- JSON path segments are escaped for dotted and special keys.
- Skill path traversal is rejected.
- Atomic writes preserve existing modes and create new files as `0600`.
- State is persisted after each successful adapter.
- Plan output is deterministic, and non-object JSON roots are rejected.
- **State adoption:** declared values already matching disk are adopted into
  state via a silent `adopt` action, so pruning and drift see pre-existing
  matching resources without rewriting user files.
- **True drift in `status`:** `status` compares each adapter's on-disk hashes
  (`ObserveHashes`) against the recorded last-applied hash, separate from
  pending desired-vs-disk config changes; a pure `homonto.toml` edit is no longer
  reported as drift.
- **Input validation:** `config.Load` rejects unknown MCP targets, empty MCP
  commands, reserved settings keys, and index-like/empty managed names.
- **Skills-only apply is link-only:** tool JSON files are written only when a
  managed key in them changes, so OpenCode JSONC comments survive link-only
  applies.
- **Doctor parity:** `doctor` checks both the Claude and OpenCode skill symlinks
  for every owned skill.
- **CI expanded:** gofmt, `go mod tidy -diff`, vet, build, test, race, stamped
  version smoke, temp-HOME CLI smoke, Docker apply smoke, and `govulncheck` all
  run in CI; the current tagged `release` workflow ships cross-platform
  `homonto` binaries, and dual-binary packaging remains release-gate work.

Known v1 product limitations, in recommended order:

Operational release blockers and hardening tasks are tracked in
[`road-to-release.md`](road-to-release.md). Do not treat this roadmap as the
release checklist.

1. **Import scope/redaction:** `import` is intentionally narrow (Claude global
   MCP servers only; env-value redaction only; command/args preserved verbatim).
   Either expand it into a fuller migration tool or keep it explicitly scoped;
   the narrow behavior is already documented in `cli-commands.md`.
2. **OpenCode JSONC comments:** any apply that touches `opencode.jsonc` rewrites
   it as normalized JSON and removes all comments. This is an accepted,
   documented limitation; comment preservation is an open question, not a bug.

Future agents should start with `/comet` (which reads `openspec/changes/` and
`.comet.yaml`) before starting v1 work.

## v1.1 Onto Framework And Catalog Projection

Homonto ships a curated bundled catalog of official frameworks and loose
resources. Frameworks are atomic bundles; loose resources can be local or
builtin. Catalog projection is explicit install behavior, not hidden runtime
magic.

Scope:

- Bundled frameworks packaged with Homonto releases: `onto`, `comet`,
  `superpowers`, and `openspec` first.
- Dependency expansion for `[frameworks.X]`, including `comet` depending on
  `superpowers` and `openspec`.
- Projection for skills, commands, and subagents into Claude Code and OpenCode
  using real tool layouts and compatibility metadata.
- Grouped `homonto plan` output for frameworks, dependencies, models,
  tool-specific projection, local/project files, conflicts, and warnings.
- `onto` binary operations backed by installed/shared framework metadata.

Non-goals:

- No remote registry in v1.1.
- No automatic updates of bundled resources after install.
- No per-resource override of framework internals.
- No implicit framework installation during `homonto init`; enabled frameworks
  remain declared in `homonto.toml` and applied through the normal plan/apply
  pipeline.

Example:

```toml
[frameworks.onto]
source = "builtin:onto"
scope = "project"

[commands.review]
source = "builtin:review"
scope = "user"
targets = ["opencode"]
```

Bundled catalog entries carry origin/version metadata for auditability. Local
adaptations live under `homonto/` and are declared with `source = "local:<name>"`.

**Status (2026-07-10, merged to `main`):** The catalog foundation for
**skills** is implemented and merged (originally
`feature/20260710/catalog-foundation-skills`). This covers the bundled
`go:embed` catalog (`onto`, `comet`, `superpowers`, `openspec` frameworks),
`[frameworks.X]` dependency expansion, version-gated materialization to
`.homonto/catalog/skills/`, and builtin SKILL projection into Claude Code and
OpenCode. Command and subagent projection (`[commands.X]`, `[subagents.X]`)
were future work as of this status; both have since landed (see the status
notes below).

Verification evidence:
- Full Go test suite: 195 tests passing across 18 packages (`go test ./...
  -count=1`), `go vet ./...` clean, `go build ./...` clean.
- Dogfood run: switching `homonto.toml` to `[frameworks.comet]` (which
  transitively pulls in `superpowers` and `openspec`) and running `homonto
  apply` materializes and links all 31 skills; a second `homonto status`
  reports `No drift.`; `homonto doctor` reports all 31 skills × 2 tools
  (Claude Code, OpenCode) = 62 "linked" OK lines.

**Status (2026-07-10, merged to `main` and archived):** Command projection
machinery has since landed on top of the skills foundation above (originally
`feature/20260710/command-projection`, now merged and its Comet change
archived).
`[commands.X]` (builtin or local, `source` resolving to `.homonto/catalog/
commands/<name>.md` or `homonto/commands/<name>.md` respectively) and
framework-declared `[commands]` tables (inherited scope/targets, transitive
through framework dependencies, deduplicated, explicit-entry collisions
rejected) both single-file-materialize to `.homonto/catalog/commands/` under
the same version gate as skills, then project into Claude Code
(`.claude/commands/<name>.md`) and OpenCode (`.opencode/command/<name>.md` or
the user-scope equivalents), with `homonto doctor` verifying both tools'
links. Real bundled command content is still deferred: exactly one
placeholder `example-command` exists (`catalog/commands/example-command.md`),
declared standalone in `homonto.toml` as `[commands.example-command] source =
"builtin:example-command"` for dogfood, matching the "Placeholder fixture
command" design constraint; the `onto` framework's catalog also lists it in
its `[commands]` table, exercised by framework-expansion unit tests.

Verification evidence:
- Full Go test suite: 215 tests passing across 19 packages (`go test ./...
  -count=1`), `go vet ./...` clean, `go build ./...` clean.
- Dogfood run: with `[commands.example-command]` declared, `homonto apply`
  materializes and links it into both tools; a follow-up `homonto status`
  reports `No drift.`, and `homonto doctor` reports `ok: command
  "example-command" linked (claude)` and `ok: command "example-command"
  linked (opencode)`.

**Status (2026-07-10, merged to `main`):** Subagent projection machinery has
since landed on top of the skills/commands foundation above (originally
`feature/20260710/subagent-projection`). `[subagents.X]` (builtin or local,
`source` resolving to `.homonto/catalog/subagents/<name>.md` or
`homonto/subagents/<name>.md` respectively) and framework-declared
`[subagents]` tables (inherited scope/targets, transitive through framework
dependencies, deduplicated, explicit-entry collisions rejected) both
single-file-materialize verbatim (no model injection) to
`.homonto/catalog/subagents/` under the same version gate as skills and
commands, then link into Claude Code (`.claude/agents/<name>.md`) and
OpenCode (`.opencode/agent/<name>.md`, or the user-scope equivalents), with
`homonto doctor` verifying both tools' links. Unlike the command projection
placeholder, subagent projection ships with real bundled content: three
subagents — `code-reviewer`, `codebase-explorer`, and comet's
`comet-navigator` — are declared in the catalog and dogfooded through
`homonto.toml`.

Verification evidence:
- Full Go test suite: 239 tests passing across 20 packages (`go test ./...
  -count=1`), `go test -race ./...` clean, `go vet ./...` clean, `go build
  ./...` clean, `gofmt -l .` clean.
- Dogfood run: with `[subagents.code-reviewer]` and
  `[subagents.codebase-explorer]` declared directly in `homonto.toml` and
  `comet-navigator` pulled in transitively via `[frameworks.comet]`, `homonto
  apply` materializes and links all three into both tools; a follow-up
  `homonto status` reports `No drift.`, and `homonto doctor` reports `ok:
  subagent "<name>" linked (claude)` and `ok: subagent "<name>" linked
  (opencode)` for all three.

## v1.2 Plugin Configuration

Plugin support expands from simple references to declarations with configuration.
Claude and OpenCode keep separate plugin schemas because their plugin systems do
not map one-to-one.

**Status (2026-07-11, first increment merged to `main`):** The plugin
*declaration model* has landed. `homonto.toml` now declares plugins as
`[plugins.<tool>.<name>]` tables with a required `source` and an optional
`enabled` flag (default true), replacing the prior bare-name lists (a breaking,
pre-release schema change). Both adapters project enable **and** disable — Claude
writes `enabledPlugins[<source>] = true|false` (disable is now expressible, not
just absence) and OpenCode adds/removes the `source` in its `plugin` array,
surgically and idempotently; a duplicate `source` across two declarations is
rejected at load to keep plans deterministic. Remaining v1.2 increments:
per-plugin `config` → Claude `pluginConfigs`; Claude marketplace registration
(`extraKnownMarketplaces`); and OpenCode `config` handling (OpenCode has no
native per-plugin config).

Scope:

- Declare plugins per target tool.
- Enable or disable plugins where the tool supports it.
- Configure plugin-specific settings.
- Show plugin config changes in `plan`.
- Apply plugin config surgically without overwriting unrelated tool settings.

Example:

```toml
[plugins.claude.claude-hud]
source = "claude-hud@official"
enabled = true
config = { compact = true, status_line = "tokens" }

[plugins.opencode.opencode-quota]
source = "@slkiser/opencode-quota"
enabled = true
config = { show_remaining = true }
```

Non-goals:

- No full marketplace, search, ratings, or community discovery in v1.2.
- No cross-tool abstraction that hides real Claude/OpenCode plugin differences.

## v1.3 Tool TUI Configuration

Homonto manages Claude/OpenCode TUI-related configuration. This phase does not
add an interactive Homonto TUI.

Scope:

- Themes and display preferences.
- Status line or model-display settings where supported.
- TUI-oriented plugins and their config.
- Keybindings or layout settings when represented in target tool config.
- Fixture-based tests for each supported target config shape.

Example:

```toml
[settings.claude.tui]
theme = "dark"
status_line = true

[settings.opencode.tui]
theme = "gruvbox"
sidebar = "auto"
```

TUI plugin configuration should live under the plugin config model when the
behavior belongs to a plugin. Tool-native UI settings should live under
`settings.<tool>.tui`.

## v2 Agent Lifecycle

Agents become first-class managed resources. v1 can link owned agent files, but
v2 manages source, version, compatibility, updates, and migration.

Scope:

- Local authored agents under `homonto/agents/`.
- Built-in agent resources from the curated catalog.
- Remote/community agent sources after local and built-in flows are stable.
- Version pinning and lockfile/state tracking.
- Compatibility checks per target tool.
- `homonto agents list`, `add`, `update`, `pin`, `doctor`, and `migrate`.
- Local-edit conflict detection before updates or migrations.
- Backup or three-way-merge behavior for lifecycle-managed agent files.

Example:

```toml
[agents.review]
source = "builtin:review-agent"
version = "1.2.0"
targets = ["claude", "opencode"]
mode = "copy"
```

Design principle: lifecycle-managed agents need stronger ownership metadata than
simple symlinked content. Homonto must be able to distinguish user-authored
agents, bundled catalog resources, and remotely sourced agents before it offers
updates or migrations.

## Data Model Principles

- Preserve the simple v1 syntax for common cases.
- Add richer table syntax only when configuration or lifecycle metadata is
  needed.
- Keep target-specific plugin schemas separate.
- Store catalog origin for auditability, but treat local provider content as
  user-owned.
- Treat full agent lifecycle as v2, not as an implicit extension of v1 symlinks.
- Resolve paths relative to the selected config file, not the shell working
  directory, so `--config` works consistently.

## Safety Rules

Every phase must preserve the v1 safety rules:

- `plan` never prints resolved secrets.
- `apply` resolves secrets only after confirmation.
- All secret resolution needed for a write succeeds before any file is changed.
- Writes are atomic.
- Managed keys are surgical.
- Unmanaged keys survive.
- Existing user-owned content is never overwritten without explicit force or
  backup behavior.
- OpenCode JSONC comments are removed whenever homonto rewrites
  `opencode.jsonc`; this is explicitly documented until comment preservation is
  implemented.
- Adapter behavior must be idempotent: a second plan after apply is no-op unless
  user-visible state changed.

## Testing Strategy

- **v1 Core:** parser, resolver, state, adapters, secret safety, idempotency,
  status/drift, import, validation, pruning, and end-to-end apply tests. CI
  should run `go test`, `go test -race`, `go vet`, `go build`, `gofmt`,
  `go mod tidy -diff`, stamped-version smoke, and temp-HOME CLI smoke tests.
- **v1.1 Onto Framework And Catalog Projection:** catalog parsing, dependency
  expansion, framework install/projection, command/subagent projection, model
  routing, conflict safety, and target compatibility tests.
- **v1.2 Plugin Configuration:** plugin config projection tests per tool,
  plugin setting diff tests, and unmanaged-key preservation tests.
- **v1.3 Tool TUI Configuration:** fixture tests for Claude/OpenCode TUI-related
  settings and TUI plugin config.
- **v2 Agent Lifecycle:** source resolution, lock/state behavior, version
  pinning, compatibility matrix, update/migration flows, and local-edit conflict
  tests.

## Open Questions

- Which bundled frameworks and loose resources should ship after the first set
  (`onto`, `comet`, `superpowers`, `openspec`).
- Whether remote framework, resource, and agent sources should share one registry
  model.
- Whether v2 agent lifecycle should use a lockfile separate from `.homonto/state.json`.
- Whether OpenCode JSONC comments should be preserved at all, or whether
  whole-file comment removal remains an accepted limitation.
- Whether import should become a full migration tool or stay a narrow Claude MCP
  bootstrap command.
- Whether status should retain a separate "pending config change" view after true
  disk-vs-state drift is implemented.
