# Dual-Binary Release Design

Date: 2026-07-09
Status: Accepted direction; implementation partial

## Summary

The first public release remains `v0.1.0-rc.1`, but its release gate changes.
The release is no longer a config-projector-only beta. It must ship a
dual-binary product:

- `homonto`: deterministic installer and config projector for AI coding tools.
- `onto`: managed spec-driven development workflow operator.

The target audience is power users and teams that know what they are doing and
want an opinionated, repeatable setup. Homonto should not optimize for beginner
onboarding, marketplaces, remote package fetching, or hidden workflow magic in
the first release.

## Product Identity

`homonto` is the foundation. It reads `homonto.toml`, expands frameworks and
loose resources, validates dependencies/model config/scopes, presents a grouped
compatibility plan, and projects resources into Claude Code and OpenCode.

`onto` is the workflow layer. It operates the spec-driven development process
declared by the `onto` framework. It creates and validates deterministic
skeletons, enforces structural workflow invariants, and records live change
state in `onto-state.yaml`. AI tool skills and agents remain responsible for
reasoning, asking questions, filling substantive content, and judging quality.

The future LXC isolation connector is intentionally out of scope for this
release. Current skills must not mention or depend on LXC behavior.

## Config Model

The previous list-style `[skills] scope` plus `[skills] own` model has been
replaced in the config loader before public release. New resource types use
explicit per-resource tables. Scope is always required; there are no hidden
`user` or `project` defaults. Framework expansion and command/subagent
projection are still pending.

Example shape:

```toml
[frameworks.onto]
source = "builtin:onto"
scope = "project"

[frameworks.comet]
source = "builtin:comet"
scope = "project"

[skills.grill-me]
source = "local:grill-me"
scope = "project"

[commands.review]
source = "builtin:review"
scope = "user"
targets = ["opencode"]

[subagents.architect]
source = "builtin:architect"
scope = "project"
targets = ["claude", "opencode"]
```

Local provider content lives under `homonto/`, resolved relative to the
directory containing `homonto.toml`. Generated state and cache live under
`.homonto/` only.

Example local provider layout:

```text
homonto/
  skills/grill-me/
  commands/review-pr.md
  subagents/architect.toml
.homonto/
  state.json
```

Installed resource names must be unique. If a user wants to adapt a bundled
loose skill, command, or subagent, they declare a local provider resource such
as `source = "local:grill-me"`. There is no special fork lifecycle in the first
release.

## Model Routing

Commands and subagents have one of three hard-assigned levels from framework or
resource metadata:

- `architectural`: design, architecture, deep analysis, and adversarial review.
- `coding`: implementation, tests, refactors, and normal engineering tasks.
- `trivial`: copy edits, small config/docs/prompt changes, and simple fixes.

Users cannot change a command or subagent level. Skills do not use levels.

All three levels are required for each enabled target tool. Each tool/level
entry requires `model` and at least one of `effort` or `variant`. `model`,
`effort`, and `variant` are validated for presence only; Homonto does not try to
maintain a catalog of valid model names or model-specific effort values.

Example:

```toml
[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"
effort = "normal"

[models.claude.trivial]
model = "haiku"
effort = "fast"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
```

The example is abbreviated. A complete config that enables OpenCode must also
define `models.opencode.coding` and `models.opencode.trivial`; the same rule
applies to every enabled target tool.

## Frameworks And Resources

Frameworks are atomic bundles. A user can enable or disable an entire framework,
but cannot remove or override individual framework internals in the first
release. Loose skills, commands, and subagents are separate installable
resources and may be adapted through the local provider.

First-release bundled frameworks:

- `onto`
- `comet`
- `superpowers`
- `openspec`

`comet` depends on `superpowers` and `openspec`. Framework and resource
dependencies are typed and declared in TOML metadata. `homonto plan` expands
transitive dependencies automatically and shows them before apply.

Allowed first-release sources:

- `builtin:<name>` for bundled catalog resources.
- `local:<name>` for project-local resources under `homonto/`.

Remote fetching, registries, marketplaces, ratings, community discovery, and a
public third-party package format are out of scope for the first release.
Bundled catalog and framework/resource metadata still need versions for
auditability and future migration work.

## Metadata Format

Machine-readable framework and resource metadata uses TOML. Markdown remains the
format for human- and agent-facing content: skills, prompts, workflow guides,
and skeleton templates.

The shared framework metadata is the source of truth for workflow operations
where possible. `homonto` uses it to install resources. `onto` reads the same
metadata for workflow structure and templates, plus `onto-state.yaml` for live
change state.

## Tool Projection

Claude Code and OpenCode are equal first-class targets. Homonto does not pretend
they have identical resource models. Each resource declares per-tool projection
metadata because tool formats differ.

A resource may target only one tool. Unsupported target/resource combinations
must be visible in `homonto plan` and must never be silently skipped.

`homonto plan` should group output by:

- Frameworks
- Dependencies
- Models
- Claude Code projection
- OpenCode projection
- Local/project files
- Conflicts and warnings

The compatibility report should include resource name, kind, source provider,
version, scope, target tools, unsupported target combinations, transitive
dependencies, effective model routing for commands/subagents, and concrete
filesystem changes.

## Apply Semantics

`homonto apply` must preflight all known conflicts before writing anything. This
includes invalid config, missing required model levels, resource name collisions,
missing secrets, unsupported resource/target combinations that are errors, and
managed file conflicts.

Known/preflightable conflicts abort before any write. Unexpected runtime adapter
failures may still use the existing fault-isolation model, as long as successful
adapter state is recorded safely.

Framework installs use a hybrid copy/symlink model:

- Native bundled framework resources are managed and repeatable, and should be
  linked into tool/project integration locations where the target tool supports
  links safely.
- Local provider resources are user-owned source content under `homonto/` and
  are projected from that source rather than treated as catalog-managed forks.
- Generated tool integrations and state are not committed.

Teams commit `homonto.toml`, `homonto/` local source content, and project docs.
They do not commit generated Claude/OpenCode integration files or
`.homonto/state.json`.

## Onto Binary

`onto` is managed by Homonto and is not an alternate installer.

Mutating `onto` commands require:

- A `homonto.toml` file.
- A declared `[frameworks.onto]` entry.
- Installed/shared framework metadata.

`onto status` is the read-only degraded exception. It may inspect existing
`docs/` artifacts without config for diagnostics and recovery.

`onto init` creates the `docs/` workflow layout, but only after the project
declares `onto` through Homonto. If the framework install is missing, `onto init`
tells the user to initialize/apply Homonto first. `homonto init` may offer
framework choices and write `homonto.toml`, but it does not create `docs/`.

First-release workflow artifacts live under:

- `docs/changes/`
- `docs/specs/`
- `docs/adr/`
- `docs/guides/`

The change state file is named `onto-state.yaml`. There is no migration or
backward-compatibility layer for the old `state.yaml` name before public
release.

`onto` creates and validates skeletons. Skills and agents fill substantive
content. The binary enforces structural invariants:

- Required files exist.
- Phase and gate records are consistent.
- Phase transitions happen only through valid gates.
- Dependencies are not unresolved.
- Archive and close rules are followed.

Dirty worktrees produce warnings for normal workflow operations and block
close/archive or release-critical operations.

## Commands And Skills

Slash commands are thin entrypoints. They route to a skill or binary operation
and do not duplicate workflow logic. Skills remain the source of agent-facing
workflow instructions.

Framework-agnostic loose skills remain supported. GitHub issue and PR helpers,
for example, should be loose resources or framework extensions later, not core
`onto` behavior in the first release.

## Doctor Boundary

`homonto doctor` checks installation and projection health:

- Config parses.
- Framework resources are installed or linked into target tools.
- Skills, commands, and subagents are present.
- Tool config files are valid and writable.
- Managed-resource conflicts are reported.

`onto doctor` checks workflow and project health:

- `docs/` layout exists.
- `onto-state.yaml` files are valid.
- Phase derivation matches artifacts.
- Gates and dependencies are consistent.
- Specs, ADRs, guides, and change archive layout are valid.

## Import

`homonto import` remains experimental. It must be documented outside the main
quickstart and must not imply support for importing frameworks, skills,
commands, subagents, or existing agent ecosystems in the first release.

## Testing And Release Gate

The first public release is blocked until the dual-binary product passes the new
gate.

Required coverage:

- Unit and CLI tests for the new explicit config model.
- Real-layout fixtures for Claude Code and OpenCode skills, commands, and
  subagents.
- Framework expansion tests for `onto`, `comet`, `superpowers`, and `openspec`.
- Dependency expansion, resource collision, and managed-conflict tests.
- Model-level validation tests for all enabled target tools.
- Docker smoke tests for user-scope and project-scope installs.
- Docker smoke tests for framework install, loose local provider install,
  command/subagent projection, conflict safety, idempotent second apply,
  `onto status`, and `onto doctor`.

## Non-Goals For First Release

- Remote framework or resource fetching.
- Public third-party package/catalog compatibility promises.
- Marketplace, search, ratings, or community discovery.
- LXC isolation connector or isolated agent execution.
- `onto` spawning Claude/OpenCode sessions directly.
- Custom user-defined workflow graphs.
- Per-subagent model overrides.
- Importing existing frameworks, skills, commands, or subagents from tool dirs.
- Individually overriding framework internals.
- Configurable `onto` artifact roots outside `docs/`.

## Open Implementation Questions

- Exact Claude Code command and subagent file formats need fixture confirmation.
- Exact OpenCode command and subagent file formats need fixture confirmation.
- Release build packaging must be updated to ship both `homonto` and `onto`.
- Current docs, skills, and tests that mention `state.yaml` must be updated to
  `onto-state.yaml` as part of the future `onto` binary implementation, not as a
  claim about current behavior.
- The config loader has migrated to explicit framework/resource tables; remaining
  work is framework/catalog expansion, command/subagent projection, and release
  packaging for both binaries.
