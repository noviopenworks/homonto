# homonto — Post-v1 Roadmap

**Date:** 2026-07-03
**Status:** Current roadmap with v1 gap list

## Summary

`homonto` v1 remains focused on the safe core: one declarative
`homonto.toml`, a plan/confirm/apply pipeline, reference-only secrets,
surgical writes, and Claude Code/OpenCode adapters. The core is implemented,
testable (129 tests across 15 packages), and the original v1 safety,
idempotency, drift, validation, and CI gaps have been closed. What remains is
either consciously accepted as a documented limitation or belongs to the
post-v1 roadmap below.

Post-v1 expands Homonto from a config projector into a manager for the AI
coding-tool ecosystem around those configs: built-in content templates, richer
plugin configuration, Claude/OpenCode TUI-related settings, and full lifecycle
management for agents.

## Roadmap Strategy

Use a layered roadmap instead of expanding v1. The current v1 implementation
plan writes into real user tool configuration, so the first milestone must prove
safety, idempotency, drift detection, and surgical merge behavior before adding
broader product surface area.

Phases:

1. **v1 Core** — existing implementation plan, with safety/idempotency fixes.
2. **v1.1 Built-In Templates** — curated official skills, agents, commands,
   and rules users can copy into their repo.
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
  version smoke, and a temp-HOME CLI smoke all run in CI.

Remaining v1 work, in recommended order:

1. **Import scope/redaction:** `import` is intentionally narrow (Claude global
   MCP servers only; env-value redaction only; command/args preserved verbatim).
   Either expand it into a fuller migration tool or keep it explicitly scoped;
   the narrow behavior is already documented in `cli-commands.md`.
2. **OpenCode JSONC comments:** any apply that touches `opencode.jsonc` rewrites
   it as normalized JSON and removes all comments. This is an accepted,
   documented limitation; comment preservation is an open question, not a bug.

Future agents should read `docs/NEXT_AGENT.md` before starting v1 work.

## v1.1 Built-In Templates

Homonto ships a curated catalog of official templates. Templates are source
material, not hidden runtime behavior.

Scope:

- Built-in skills, agents, commands, and rules packaged with Homonto releases.
- `homonto templates list` shows available templates with type, description,
  version, and target-tool compatibility.
- `homonto templates add <name>` copies selected templates into `content/`.
- Copied templates become user-owned content and may be edited freely.
- Existing local content is not overwritten unless the user passes an explicit
  force or backup option.

Non-goals:

- No remote registry in v1.1.
- No automatic template updates after copy.
- No implicit template installation during `homonto init`.

Example:

```toml
[templates]
enabled = ["graphify-skill", "review-agent"]
```

The `enabled` list records template origin for auditability. The actual content
lives under `content/` and remains the user's copy.

## v1.2 Plugin Configuration

Plugin support expands from simple references to declarations with configuration.
Claude and OpenCode keep separate plugin schemas because their plugin systems do
not map one-to-one.

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

- Local authored agents under `content/agents/`.
- Built-in agent templates from the curated template catalog.
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
agents, copied built-in templates, and remotely sourced agents before it offers
updates or migrations.

## Data Model Principles

- Preserve the simple v1 syntax for common cases.
- Add richer table syntax only when configuration or lifecycle metadata is
  needed.
- Keep target-specific plugin schemas separate.
- Store template origin for auditability, but treat copied content as user-owned.
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
- **v1.1 Templates:** catalog parsing, copy/no-overwrite behavior, template
  validation, and target compatibility tests.
- **v1.2 Plugin Configuration:** plugin config projection tests per tool,
  plugin setting diff tests, and unmanaged-key preservation tests.
- **v1.3 Tool TUI Configuration:** fixture tests for Claude/OpenCode TUI-related
  settings and TUI plugin config.
- **v2 Agent Lifecycle:** source resolution, lock/state behavior, version
  pinning, compatibility matrix, update/migration flows, and local-edit conflict
  tests.

## Open Questions

- Which built-in templates should ship first.
- Whether remote template and agent sources should share one registry model.
- Whether v2 agent lifecycle should use a lockfile separate from `.homonto/state.json`.
- Whether OpenCode JSONC comments should be preserved at all, or whether
  whole-file comment removal remains an accepted limitation.
- Whether import should become a full migration tool or stay a narrow Claude MCP
  bootstrap command.
- Whether status should retain a separate "pending config change" view after true
  disk-vs-state drift is implemented.
