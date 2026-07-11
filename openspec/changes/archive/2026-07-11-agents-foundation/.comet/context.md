# Comet Design Handoff

- Change: agents-foundation
- Phase: design
- Mode: compact
- Context hash: 6fa8ca5482b7ac631338d502b011ec7548645a1c75e376d7bcd19370202abdea

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-foundation/proposal.md

- Source: openspec/changes/agents-foundation/proposal.md
- Lines: 1-57
- SHA256: 99364f64de7065311ecc05e5d2b5266bd655b81ae7cd23ecfc13579ae50e5e0a

```md
## Why

Roadmap v2 (Agent Lifecycle) makes agents first-class managed resources with
source, version, compatibility, updates, and migration — a package-manager for
agents, distinct from v1's simple symlinked `[subagents.<name>]` files. The
roadmap's own design principle: "Treat full agent lifecycle as v2, not as an
implicit extension of v1 symlinks," and lifecycle-managed agents "need stronger
ownership metadata." This change is v2's foundation increment: the
`[agents.<name>]` declaration model and a read-only `homonto agents list` — no
lifecycle mutation yet (add/update/pin/migrate, lockfile, compatibility checks,
three-way-merge, and remote sources are deferred to later increments, exactly as
the onto binary started with a read-only `status`).

## What Changes

- Add the `[agents.<name>]` config model: `type Agent { Source string; Version
  string; Targets []string; Mode string }` and `Config.Agents map[string]Agent`.
  ```toml
  [agents.review]
  source  = "builtin:review-agent"   # builtin:<name> | local:<name> (remote deferred)
  version = "1.2.0"                   # optional; empty = unpinned
  targets = ["claude", "opencode"]    # optional; default both
  mode    = "copy"                    # optional; copy | link (default link)
  ```
- **Validation**: the agent name is a valid config key; `source` uses the
  existing `builtin:<name>` / `local:<name>` scheme (remote schemes rejected for
  now); `targets` ∈ {claude, opencode}; `mode` ∈ {copy, link} (empty → link).
- Add `homonto agents list`: a read-only command that loads the config and prints
  each declared agent (sorted): name, source, version (or `unpinned`), targets,
  and mode. It performs no projection and no mutation. `homonto agents` with no
  subcommand shows help.
- This is additive and independent of the existing `[subagents.<name>]`
  projection — no v1 behavior changes.

## Capabilities

### New Capabilities

- `agent-lifecycle`: `homonto agents list` reports declared lifecycle-managed
  agents read-only. (Mutation commands — add/update/pin/doctor/migrate — and the
  lockfile arrive in later increments.)

### Modified Capabilities

- `config-model`: adds the `[agents.<name>]` declaration (source/version/targets/
  mode) with validation.

## Impact

- `internal/config/config.go`: `Agent` type (+ `TargetsOrAll`/`ModeOrDefault`
  helpers), `Config.Agents`, `validateAgents`.
- `internal/cli/`: new `agents.go` (`agentsCmd()` parent + `list` subcommand),
  registered on the root.
- Tests in `internal/config` and `internal/cli`.
- No new dependency. No projection/adapter/state change (read-only foundation).
- Establishes the v2 agent-lifecycle surface; later increments add mutation, the
  lockfile, compatibility checks, and remote sources.

```

## openspec/changes/agents-foundation/design.md

- Source: openspec/changes/agents-foundation/design.md
- Lines: 1-71
- SHA256: 9a80490b6ea9e9ae1913ef37b605abea522d3452ec9569248e5be2c3337ea7e1

```md
## Context

Roadmap v2 foundation. Agents become first-class managed resources with
lifecycle metadata (version, mode) beyond v1's `[subagents.<name>]` symlinks.
This increment adds only the declaration model + a read-only `homonto agents
list`, deferring all mutation (add/update/pin/migrate), the lockfile,
compatibility checks, three-way-merge, and remote sources — mirroring how the
onto binary started read-only.

## Goals / Non-Goals

**Goals**: `[agents.<name>]` model (`Agent{Source,Version,Targets,Mode}`) +
validation reusing the existing `validSource`/`validateKey`/target checks; a
read-only `homonto agents list`.

**Non-Goals (this increment)**: any projection or file write for agents; the
lockfile/state; `add`/`update`/`pin`/`doctor`/`migrate`; compatibility checks;
three-way-merge/backup; remote sources; changing `[subagents.<name>]`.

## Decisions

### D1 — Model (`internal/config/config.go`)

```go
type Agent struct {
    Source  string   `toml:"source"`
    Version string   `toml:"version"`
    Targets []string `toml:"targets"`
    Mode    string   `toml:"mode"`
}
func (a Agent) TargetsOrAll() []string { if len(a.Targets)==0 { return []string{"claude","opencode"} }; return a.Targets }
func (a Agent) ModeOrDefault() string  { if a.Mode=="" { return "link" }; return a.Mode }
// Config gains: Agents map[string]Agent `toml:"agents"`
```

### D2 — Validation (`validateAgents`, called from Parse/Load)

For each `name, ag := range c.Agents`: `validateKey("agents", name)`;
`validSource(ag.Source)` (the existing builtin:/local: check — reject remote/
unknown); `ag.Mode ∈ {"", "copy", "link"}` else error naming agent+mode; each
target ∈ {claude, opencode}. Reuse the exact error-message style of
`validateResources`.

### D3 — `homonto agents list` (`internal/cli/agents.go`)

A parent `agentsCmd()` (Use `agents`, no RunE → shows help) with a `list`
subcommand. `list` reads `--config`, `config.Load(cfgPath)`, sorts agent names,
and prints one line per agent:
`<name>: <source>  version=<v|unpinned>  targets=<claude,opencode>  mode=<link|copy>`.
Empty → `No agents declared.`. Register `agentsCmd()` on the root next to the
other commands. Read-only: loads config only, never builds the engine or writes.

## Risks / Trade-offs

- **Model vs `[subagents]` overlap**: both describe agents, but `[subagents]` is
  the v1 symlink `Resource` (scope-based) and `[agents]` is the v2 lifecycle
  model (version/mode-based). They coexist; a later increment decides whether
  `[agents]` supersedes `[subagents]`. This increment keeps them independent and
  documents the distinction.
- **Read-only list of unrealized agents**: `list` shows declared intent, not
  installed state (no lockfile yet). The output labels are about declaration; a
  later `doctor`/`status` increment adds installed/version state.

## Migration Plan

Additive; `[agents]` optional. No migration.

## Open Questions

None for the foundation. Whether `[agents]` eventually subsumes `[subagents]` is
a later-increment decision.

```

## openspec/changes/agents-foundation/tasks.md

- Source: openspec/changes/agents-foundation/tasks.md
- Lines: 1-17
- SHA256: 14ddd13d5a1fe22bbbfcad251ce3fe95c26e10e5c5fb8c87b539ad223153f6a9

```md
## 1. Config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) Add `type Agent { Source, Version string; Targets []string; Mode string }` (toml tags) + `TargetsOrAll()` + `ModeOrDefault()`; add `Agents map[string]Agent `toml:"agents"`` to `Config`.
- [ ] 1.2 (TDD RED first) `validateAgents`: `validateKey("agents",name)`; `validSource(source)` (builtin:/local:); `mode ∈ {"",copy,link}` else error naming agent+mode; targets ∈ {claude,opencode}. Call from Parse/Load. Tests: parse full agent; defaults (version empty→unpinned, targets empty→both, mode empty→link); invalid source (https) rejected; invalid mode (symlink) rejected; unknown target rejected.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [agents.<name>] lifecycle declaration model + validation`

## 2. `homonto agents list` (`internal/cli`)

- [ ] 2.1 (TDD RED first) Add `internal/cli/agents.go`: `agentsCmd()` parent (Use `agents`, shows help) + `list` subcommand that `config.Load(--config)`, sorts agent names, prints `<name>: <source>  version=<v|unpinned>  targets=<...>  mode=<...>` per agent (or `No agents declared.`). Read-only. Register `agentsCmd()` on the root.
- [ ] 2.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","list","--config",path])`: two agents → both printed sorted with source/version/targets/mode; no `[agents]` → `No agents declared.`; unpinned agent shows `unpinned`; read-only (no files written — the command only loads config).
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents list' (read-only) reports declared agents`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a config with two `[agents.<name>]` (one pinned, one unpinned, one local mode=copy) → `homonto agents list` prints them sorted; an invalid agent source fails load.
- [ ] 3.2 Update `docs/roadmap.md` v2 status (foundation: `[agents.<name>]` model + read-only `agents list` landed; mutation/lockfile/remote deferred) + README `[agents.<name>]` example. No over-claim (no lifecycle mutation yet).
- [ ] 3.3 Commit all changes.

```

## openspec/changes/agents-foundation/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-foundation/specs/agent-lifecycle/spec.md
- Lines: 1-26
- SHA256: ce003c6b123dd7c7917028044f65a824d92c2240fadbd7c62358da46d2d3f502

```md
## ADDED Requirements

### Requirement: homonto agents list reports declared agents

`homonto agents list` SHALL be a read-only command that loads the config
(honoring `--config`) and prints each declared `[agents.<name>]` agent, sorted by
name, showing its name, source, version (or an unpinned indicator), targets, and
mode. It SHALL perform no projection and no mutation. When no agents are declared
it SHALL say so. `homonto agents` with no subcommand SHALL show help.

#### Scenario: List declared agents

- **GIVEN** a config with two `[agents.<name>]` agents
- **WHEN** `homonto agents list` runs
- **THEN** it prints both agents sorted by name, each with source, version-or-unpinned, targets, and mode, and exits 0

#### Scenario: No agents declared

- **GIVEN** a config with no `[agents]` section
- **WHEN** `homonto agents list` runs
- **THEN** it reports that no agents are declared and exits 0

#### Scenario: agents list is read-only

- **WHEN** `homonto agents list` runs
- **THEN** it writes no files and mutates no tool config or state

```

## openspec/changes/agents-foundation/specs/config-model/spec.md

- Source: openspec/changes/agents-foundation/specs/config-model/spec.md
- Lines: 1-40
- SHA256: 1ce9772d5d028df709e63472a409f1308febd003eca5bbf921fc2f2dd44c2048

```md
## ADDED Requirements

### Requirement: Agent lifecycle declaration

Lifecycle-managed agents SHALL be declarable as `[agents.<name>]` tables, distinct
from the v1 `[subagents.<name>]` symlink model. Each agent table SHALL carry:

- `source` (required): the agent source, using the `builtin:<name>` or
  `local:<name>` scheme (remote schemes are not yet accepted);
- `version` (optional string): a pinned version; empty means unpinned;
- `targets` (optional list): target tools ∈ {`claude`, `opencode`}; empty means
  both;
- `mode` (optional): `copy` or `link`; empty defaults to `link`.

The agent name SHALL be validated as a config key. An invalid source scheme, an
unknown target, or an invalid `mode` SHALL be rejected at load naming the agent.

#### Scenario: Parse an agent declaration

- **GIVEN** `[agents.review]` with `source = "builtin:review-agent"`, `version = "1.2.0"`, `targets = ["claude","opencode"]`, `mode = "copy"`
- **WHEN** the config is parsed
- **THEN** it yields an agent `review` with that source, version, targets, and mode

#### Scenario: Defaults for optional fields

- **GIVEN** `[agents.x]` with only `source = "local:x"`
- **WHEN** the config is parsed
- **THEN** the agent has empty version (unpinned), both tools as targets, and mode `link`

#### Scenario: Invalid agent source is rejected

- **GIVEN** `[agents.x]` with `source = "https://example.com/x"`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the agent and the invalid source

#### Scenario: Invalid agent mode is rejected

- **GIVEN** `[agents.x]` with `source = "builtin:x"` and `mode = "symlink"`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the agent and the invalid mode

```
