# Comet Design Handoff

- Change: homonto-v1-core
- Phase: design
- Mode: compact
- Context hash: b5752c3ef783f763b6e630fb10762110e09b6d01d3cf75953f0ba461e9bb93f3

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/homonto-v1-core/proposal.md

- Source: openspec/changes/homonto-v1-core/proposal.md
- Lines: 1-63
- SHA256: 03b5b43eba839b8c4010d8789a59f640e5590a963adb29fcda51923155c533d2

```md
## Why

AI coding-tool configuration (MCP servers, skills, plugins, settings) is scattered
across per-tool files (`~/.claude.json`, `~/.claude/settings.json`,
`~/.config/opencode/opencode.jsonc`), edited by hand, and impossible to reproduce
or review. `homonto` v1 makes one declarative `homonto.toml` the single source of
truth and projects it into Claude Code and OpenCode through a terraform-style
plan/confirm/apply pipeline, with secrets referenced (never stored) and resolved
only at apply time. This is the foundation every post-v1 roadmap phase builds on,
so it must first prove safety, idempotency, drift detection, and surgical merge.

## What Changes

- New Go CLI `homonto` (module `github.com/noviopenworks/homonto`, Go 1.22+) with
  commands `init`, `import`, `plan`, `apply`, `status`, `doctor`.
- Parse `homonto.toml` into one tool-agnostic desired-state model (MCPs, owned
  skills, per-tool plugins, per-tool settings).
- Per-tool **adapters** (Claude Code, OpenCode) that `Read`/`Plan`/`Apply` via
  **surgical merge**: homonto writes only the keys it manages and preserves all
  unmanaged keys (and, where possible, JSONC comments) in each tool's file.
- **Reference-only secrets** (`${pass:…}`, `${ENV}`) resolved **after** confirm,
  **all-at-once before any write** (two-phase); an interrupted or under-resolved
  apply never leaves a half-written file (atomic temp+rename; state written last).
- Owned content (skills) linked into each tool via **symlinks**, with conflict
  detection (never clobber a non-managed file).
- **Secret-idempotency fix** (roadmap-required pre-implementation adjustment):
  state stores each managed key's *unresolved* desired value plus a *non-secret
  hash* (sha256) of the applied resolved value. A second `plan` on a secret-backed
  value is a no-op, drift of a secret value is still detected, and neither `plan`
  output nor `state.json` ever contains a plaintext secret. **BREAKING** vs. the
  original plan's naive unresolved-only state comparison.
- Local state at `<repo>/.homonto/state.json` (gitignored) for drift detection.

## Capabilities

### New Capabilities
- `config-model`: parsing `homonto.toml` into the tool-agnostic desired-state
  model, including target defaulting (an MCP with no `targets` applies to all).
- `apply-pipeline`: the six-stage plan → confirm → resolve → apply engine —
  two-phase secret handling, atomic writes, idempotent re-apply, and drift.
- `secret-references`: `${pass:…}`/`${ENV}` resolution timing and the hashed-state
  idempotency model that keeps `plan` output and `state.json` free of plaintext.
- `tool-adapters`: Claude Code and OpenCode projection of MCPs/settings/plugins via
  surgical JSON/JSONC merge, plus symlinked owned content with conflict detection.
- `cli-commands`: `init`, `import`, `plan`, `apply`, `status`, `doctor` surfaces
  and their safety behaviors (import secret redaction, no-overwrite guards).

### Modified Capabilities
- (none — greenfield; no existing specs in `openspec/specs/`.)

## Impact

- New codebase under `internal/` (`config`, `secret`, `state`, `jsonutil`, `link`,
  `adapter/{claude,opencode}`, `engine`, `cli`, `scaffold`, `importer`) plus
  `main.go`, `go.mod`, `README.md`, `.gitignore`.
- Dependencies: `spf13/cobra`, `pelletier/go-toml/v2`, `tidwall/sjson`+`gjson`,
  `tailscale/hujson`; standard `testing` + `crypto/sha256`.
- Runtime effects on the user's machine: writes to `~/.claude.json`,
  `~/.claude/settings.json`, `~/.config/opencode/opencode.jsonc`, and symlinks
  under each tool's `skills/` dir — all surgical and confirmation-gated.
- Supersedes the state-comparison approach in
  `docs/superpowers/plans/2026-06-24-homonto.md` (Tasks 4, 8, 9, 11) with the
  hashed-state idempotency model.
```

## openspec/changes/homonto-v1-core/design.md

- Source: openspec/changes/homonto-v1-core/design.md
- Lines: 1-85
- SHA256: d367542810f447f740e46c8b28d5f589bd9a9825dc326ce2bef240d8395242bc

[TRUNCATED]

```md
## Context

`homonto` v1 is a personal Go CLI: the single declarative source of truth for AI
coding-tool config, projecting `homonto.toml` into Claude Code and OpenCode. The
detailed technical RFC and the six-stage pipeline live in
`docs/superpowers/specs/2026-06-24-homonto-design.md`; a task-by-task TDD plan
exists in `docs/superpowers/plans/2026-06-24-homonto.md`. This document records
the **high-level architecture decisions** for the change; the deep Design Doc and
delta capability specs are produced in the Comet design phase.

## Architecture

Normalized desired-state model + per-tool adapters with shared services:

```
homonto.toml ──▶ Parse ──▶ DesiredState ──▶ [ ClaudeAdapter, OpenCodeAdapter ]
                                                   │ Read → Plan → Apply
shared: SecretResolver · ContentLinker · Planner/Printer · StateStore
```

Everything downstream operates on the tool-agnostic `Config`/`DesiredState`, never
on raw TOML. Adding a tool later = implement one `Adapter`, no engine changes.

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Core model | Declarative source of truth; tools are generated outputs | Reproducible, reviewable, one source |
| Secrets | Reference-only (`${pass:…}` primary, `${ENV}` fallback), resolve after confirm, all-at-once before any write | Plans/logs safe to share; no half-apply |
| Merge safety | Surgical: write only managed keys, preserve unmanaged | Never destroy a user's hand-tuned config |
| Writes | Atomic temp+rename; `state.json` written last | Crash-safe; next apply reconciles |
| Owned content | Symlinks (not copies) in v1 | Edit once, live everywhere; `--copy` later |
| Config format / stack | TOML; cobra, go-toml/v2, sjson/gjson, hujson | Matches design spec |
| **Idempotency of secrets** | State stores `{desired: <unresolved token>, applied: sha256(resolved)}` per managed key | Roadmap-required; keeps repeat plans no-op, detects drift, never stores/prints plaintext |

## Secret-idempotency model (the change's core design point)

Problem: the original plan compared the *unresolved* desired value against the
*resolved* on-disk value, so any secret-backed key showed a spurious `~ update`
every run, and a drift/update would have printed the resolved secret into `Change.Old`.

Resolution — per managed key, at plan time:

```
disk absent                                                   → create
desired has NO secret ref:  disk == desired ? noop : update   (direct compare)
desired HAS secret ref:     in-state
                            && state.desired == desired
                            && state.applied == sha256(disk)  ? noop : update
```

- On `apply`, after resolving each change's value, store
  `state.Set(tool, key, {desired: unresolved, applied: sha256(resolved)})`.
- For **any** change on a secret-bearing key, `Change.Old` is redacted (`«secret»`),
  never the on-disk resolved value — so `plan` output stays plaintext-free.
- `state.json` holds only unresolved tokens + hashes → safe to share.

## Alternatives considered

- **Token-match only, no hashing** — simpler, but loses drift detection for
  secret-backed values (can't tell if the on-disk key changed). Rejected: the
  hash is cheap and the roadmap explicitly asks for drift-safe idempotency.
- **Resolve secrets at plan time to compare** — violates "plan never touches
  `pass` / never resolves". Rejected.
- **Store plaintext resolved value in state** — violates "nothing secret in the
  repo/state; `state.json` is shareable". Rejected.

## Scope boundaries

In: config model, apply pipeline (two-phase, atomic, idempotent, drift), secret
references + hashed state, Claude + OpenCode adapters (surgical + symlinks), the 6
CLI commands, and the full test matrix (unit/golden/e2e/secret-safety/idempotency).

Out (roadmap v1.1+): built-in templates, richer plugin configuration, tool TUI
configuration, agent lifecycle; encrypted in-repo secrets; imperative add/remove;
tools beyond Claude Code + OpenCode; preserving JSONC comments inside rewritten
regions (documented caveat, not a goal).

## Risks

```

Full source: openspec/changes/homonto-v1-core/design.md

## openspec/changes/homonto-v1-core/tasks.md

- Source: openspec/changes/homonto-v1-core/tasks.md
- Lines: 1-36
- SHA256: a25c8f7d356b675acfdb11a90b4ad2e600386befcbecd8a52eb78f5ece2cd376

```md
# Tasks — homonto-v1-core

TDD throughout: write the failing test first, then minimal implementation, commit
per task. Detailed per-step guidance is refined in the build phase from
`docs/superpowers/plans/2026-06-24-homonto.md`, adjusted for the hashed-state
idempotency model (⚑ marks deltas from that plan).

## 1. Foundation
- [ ] 1.1 Scaffold module + `version` command (`go.mod`, `main.go`, `internal/cli/root.go`, `.gitignore`)
- [ ] 1.2 Config model + TOML loader (`internal/config`) — MCPs, Skills, Plugins, Settings, `TargetsOrAll`
- [ ] 1.3 Secret resolver (`internal/secret`) — `${pass:…}` + `${ENV}`, `Resolve`, `ContainsRef`
- [ ] 1.4 ⚑ Hash helper — `sha256` of a resolved value (in `internal/secret` or `internal/state`)

## 2. State + merge primitives
- [ ] 2.1 ⚑ State store with hashed entries (`internal/state`) — `Entry{Desired, Applied}`; `Set(tool,key,desired,appliedHash)`, `Get`, atomic `Save`/`Load`
- [ ] 2.2 Surgical JSON/JSONC merge (`internal/jsonutil`) — `SetJSON`, `GetJSON`, `Standardize`, `EnsureArrayElem`
- [ ] 2.3 Content linker (`internal/link`) — idempotent symlink + conflict detection (never clobber)

## 3. Adapters
- [ ] 3.1 Adapter interface + `Change`/`ChangeSet` + plan printer (`internal/adapter`, `internal/plan`) — `+`/`~`, hide noops, never resolve secrets
- [ ] 3.2 ⚑ Claude adapter (`internal/adapter/claude`) — MCP/settings/plugins surgical projection; state-aware noop for secret keys; **redact `Change.Old` for secret-bearing keys**; store `{desired, sha256(resolved)}` on apply
- [ ] 3.3 ⚑ OpenCode adapter (`internal/adapter/opencode`) + Claude skill linking — JSONC merge, plugin array append, same hashed-state + redaction rules
- [ ] 3.4 ⚑ Secret-safety tests — `plan` output **and** `state.json` never contain a resolved secret, including on **drift of a secret-backed key**

## 4. Engine + CLI
- [ ] 4.1 Engine + `plan`/`apply` (`internal/engine`, `internal/cli`) — two-phase (resolve all, abort before any write), confirm `[y/N]`/`--yes`, save state last
- [ ] 4.2 `status` (drift) + `doctor` (`pass` on PATH, tool dirs, owned-skill presence)
- [ ] 4.3 `init` scaffold (never overwrite existing files)
- [ ] 4.4 `import` — bootstrap `homonto.toml` from existing setup with **secret redaction** to `${pass:…}`; `--force` guard

## 5. Verification
- [ ] 5.1 ⚑ End-to-end test: `init`→edit→`plan`→`apply` projects into both tools + symlinks; **second apply is a no-op including a secret-backed MCP**
- [ ] 5.2 Two-phase abort test — missing secret ref → no file written, missing ref named
- [ ] 5.3 Golden-file surgical-merge tests — unmanaged keys survive in all target files
- [ ] 5.4 README (quickstart, secret-reference syntax, JSONC comment caveat, symlinked content)
- [ ] 5.5 Full suite green: `go test ./... && go vet ./... && go build ./...`
```

## openspec/changes/homonto-v1-core/specs/apply-pipeline/spec.md

- Source: openspec/changes/homonto-v1-core/specs/apply-pipeline/spec.md
- Lines: 1-70
- SHA256: 8b8588d6543e7b4d8d3b1424a95784f880b06cedcb7a67cba8f6e9528325be9a

```md
## ADDED Requirements

### Requirement: Plan is a pure dry run

`homonto plan` SHALL compute and print the diff between desired and current state
without writing any file, resolving any secret, or contacting the secret backend.

#### Scenario: Plan writes nothing
- **WHEN** the user runs `homonto plan`
- **THEN** a terraform-style diff is printed and no tool file, symlink, or state
  file is created or modified

#### Scenario: Plan shows creates and updates, hides noops
- **WHEN** the plan contains create, update, and noop changes
- **THEN** the output shows `+` for creates and `~` for updates and omits noops

### Requirement: Apply is confirmation-gated and two-phase

`homonto apply` SHALL print the plan, require confirmation (`[y/N]`, skippable
with `--yes`), and then apply in two phases: resolve **all** secrets for confirmed
changes first, and only if every resolution succeeds proceed to write. State SHALL
be saved last.

#### Scenario: Confirmation declined
- **WHEN** the user answers anything other than `y`
- **THEN** apply aborts and no file is written

#### Scenario: Missing secret aborts before any write
- **WHEN** a confirmed change references a secret that cannot be resolved
- **THEN** apply aborts before writing any file, names the missing reference, and
  leaves every tool file and the state file unchanged

### Requirement: Atomic writes

Every file write SHALL go through a temp file followed by rename, so an
interrupted apply never leaves a half-written file; `state.json` SHALL be written
after all tool files.

#### Scenario: Crash-safety ordering
- **WHEN** apply writes multiple tool files and then state
- **THEN** each tool file is individually valid at all times and state is written
  only after all tool files succeed

### Requirement: Idempotent re-apply

A second `plan` or `apply` with unchanged config and unchanged on-disk state SHALL
report no changes and SHALL NOT touch any file or the secret backend, including
for secret-backed values.

#### Scenario: Second apply is a no-op
- **WHEN** apply runs twice with no config change
- **THEN** the second run prints "No changes" and modifies nothing

#### Scenario: Secret-backed value stays idempotent
- **WHEN** a config value is a secret reference that was already applied
- **THEN** the next plan reports it as a noop (no spurious update) without
  re-resolving the secret

### Requirement: Drift detection

`homonto status` SHALL report managed keys whose on-disk value diverges from the
last-applied snapshot recorded in state.

#### Scenario: Out-of-band change surfaces
- **WHEN** a managed key is changed on disk outside homonto after an apply
- **THEN** `status` lists that key as drifted

#### Scenario: No drift after clean apply
- **WHEN** no on-disk managed value has changed since the last apply
- **THEN** `status` reports no drift
```

## openspec/changes/homonto-v1-core/specs/cli-commands/spec.md

- Source: openspec/changes/homonto-v1-core/specs/cli-commands/spec.md
- Lines: 1-50
- SHA256: c0a04f7008363d417b50d0c3a34099e4f9d469f7413063bee43e63da20680273

```md
## ADDED Requirements

### Requirement: Command surface

`homonto` SHALL expose `version`, `init`, `import`, `plan`, `apply`, `status`, and
`doctor`, with a persistent `--config` flag (default `homonto.toml`). Config
changes happen by editing `homonto.toml`; there SHALL be no imperative
`add`/`remove` mutators in v1.

#### Scenario: Version prints the build version
- **WHEN** the user runs `homonto version`
- **THEN** it prints `homonto <version>`

### Requirement: init scaffolds without overwriting

`homonto init [dir]` SHALL scaffold a starter repo (`homonto.toml`, `.gitignore`,
`.env.example`, `content/skills/`) and SHALL never overwrite an existing file.

#### Scenario: Existing files are preserved
- **WHEN** `homonto.toml` already exists in the target dir
- **THEN** `init` leaves it unchanged and only creates the missing files

### Requirement: import bootstraps with secret redaction

`homonto import` SHALL read the current Claude/OpenCode setup into a starter
`homonto.toml`, replacing any value that looks like a literal secret with a
`${pass:…}` reference and reporting a warning. It SHALL refuse to overwrite an
existing config unless `--force` is given. No literal secret SHALL be written to
the generated config.

#### Scenario: Literal secret is redacted
- **WHEN** an imported env value looks like a secret (e.g. `sk-…`, or a
  `*_KEY`/`*_TOKEN` key with a non-reference value)
- **THEN** it is replaced with a `${pass:…}` reference, a warning is emitted, and
  the literal secret never appears in the output

#### Scenario: Overwrite guarded
- **WHEN** a config already exists and `--force` is not given
- **THEN** import refuses and reports, leaving the existing config unchanged

### Requirement: doctor health checks

`homonto doctor` SHALL check that `pass` is on `PATH`, that each target tool's
config location is present, and that each owned skill exists under
`content/skills/`, reporting `ok`/`warn` lines.

#### Scenario: Missing owned skill is flagged
- **WHEN** a skill listed in `[skills] own` has no directory under
  `content/skills/`
- **THEN** `doctor` reports a warning naming that skill
```

## openspec/changes/homonto-v1-core/specs/config-model/spec.md

- Source: openspec/changes/homonto-v1-core/specs/config-model/spec.md
- Lines: 1-41
- SHA256: bd64d8e534cf377fad6cc7d32453c52d7d45022560e75096c85055efe727067f

```md
## ADDED Requirements

### Requirement: Declarative config as single source of truth

`homonto` SHALL parse a single `homonto.toml` file into one tool-agnostic
desired-state model covering MCP servers, owned skills, per-tool plugins, and
per-tool settings. All downstream stages SHALL operate on this model, never on
raw TOML.

#### Scenario: Parse a complete config
- **WHEN** `homonto.toml` declares MCP servers, `[skills] own`, per-tool
  `[plugins]`, and per-tool `[settings]`
- **THEN** the loader returns a model exposing each MCP's command/env/targets,
  the owned skill list, the per-tool plugin lists, and the per-tool settings maps

#### Scenario: Missing config file is an error
- **WHEN** the config path does not exist
- **THEN** `Load` returns an error rather than an empty model

### Requirement: MCP target defaulting

An MCP server declared without an explicit `targets` list SHALL apply to all
supported tools; an MCP with an explicit `targets` list SHALL apply only to those
tools.

#### Scenario: No targets means all tools
- **WHEN** an MCP entry omits `targets`
- **THEN** its effective targets are `["claude", "opencode"]`

#### Scenario: Explicit targets are honored
- **WHEN** an MCP entry sets `targets = ["claude"]`
- **THEN** its effective targets are exactly `["claude"]`

### Requirement: Secret references preserved as unresolved tokens

The config model SHALL retain secret references (`${pass:…}`, `${ENV}`) verbatim
as unresolved tokens; parsing SHALL NOT resolve them.

#### Scenario: Env value with a pass reference
- **WHEN** an MCP `env` value is `"${pass:ai/brave}"`
- **THEN** the parsed model stores `"${pass:ai/brave}"` unchanged
```

## openspec/changes/homonto-v1-core/specs/secret-references/spec.md

- Source: openspec/changes/homonto-v1-core/specs/secret-references/spec.md
- Lines: 1-55
- SHA256: 53bd15fd9df22b9bbe3c2d79a6d966a85c97f6119d04b8fa122cad1bd2449767

```md
## ADDED Requirements

### Requirement: Secrets are referenced, never stored

Secret values SHALL be expressed in `homonto.toml` only as references
(`${pass:PATH}` resolved via `pass`, or `${ENV}` resolved from the environment).
Plaintext secret values SHALL never be required in the repo.

#### Scenario: Pass reference resolves at apply
- **WHEN** a value is `${pass:ai/brave}` and apply is confirmed
- **THEN** the resolver invokes the `pass` backend for `ai/brave` and substitutes
  the returned value only into the file being written

#### Scenario: Env reference resolves from environment
- **WHEN** a value is `${BRAVE_API_KEY}` and the variable is set
- **THEN** the resolver substitutes the environment value

#### Scenario: Missing reference errors by name
- **WHEN** a referenced env var is unset or a `pass` path is absent
- **THEN** resolution fails with an error naming the missing reference

### Requirement: Plan output never contains a resolved secret

Plan and log output SHALL display secret-bearing values only as their unresolved
tokens, never as resolved plaintext — including when a secret-backed key is
created, updated, or reported as drifted.

#### Scenario: Create shows the token
- **WHEN** a plan creates a key whose value is `${pass:ai/brave}`
- **THEN** the output contains `${pass:ai/brave}` and never the resolved value

#### Scenario: Drift of a secret value is redacted
- **WHEN** a secret-backed key has drifted on disk and the plan shows an update
- **THEN** the change's old value is redacted (e.g. `«secret»`) and the resolved
  on-disk secret never appears in the output

### Requirement: State stores unresolved token plus a non-secret hash

For each managed key, state SHALL store the unresolved desired value and a
non-secret hash (sha256) of the resolved value written to disk. `state.json` SHALL
NOT contain any plaintext secret and SHALL remain safe to share.

#### Scenario: State records desired token and applied hash
- **WHEN** a secret-backed change is applied
- **THEN** state stores the `${pass:…}` token and `sha256(resolved value)`, not the
  resolved value

#### Scenario: Idempotency decision uses token match plus hash
- **WHEN** planning a secret-backed key that is present in state
- **THEN** it is a noop only if the desired token matches state and
  `sha256(on-disk value)` matches the stored hash; otherwise it is an update

#### Scenario: State file has no plaintext secret
- **WHEN** `state.json` is read after any apply
- **THEN** it contains no resolved secret value
```

## openspec/changes/homonto-v1-core/specs/tool-adapters/spec.md

- Source: openspec/changes/homonto-v1-core/specs/tool-adapters/spec.md
- Lines: 1-59
- SHA256: 8c58e3b28dcf3a7c5e205a556e490bc175aad557a3549afd740c7802998ebde6

```md
## ADDED Requirements

### Requirement: Surgical merge preserves unmanaged keys

Each adapter SHALL write only the keys homonto manages and SHALL preserve all
unmanaged keys already present in a tool's file. A tool file that cannot be parsed
SHALL cause that adapter to abort and report, never to overwrite.

#### Scenario: Unmanaged keys survive apply
- **WHEN** a tool file contains keys homonto does not manage
- **THEN** those keys are byte-preserved (values intact) after apply

#### Scenario: Unparseable file is not clobbered
- **WHEN** an existing tool file cannot be parsed
- **THEN** that adapter aborts and reports and does not write the file, while
  other tools still proceed

### Requirement: Claude Code projection

The Claude adapter SHALL project MCP servers into `~/.claude.json`
(`mcpServers.<name>`), settings and plugins into `~/.claude/settings.json`, and
owned skills as symlinks under `~/.claude/skills/`.

#### Scenario: MCP and setting projected surgically
- **WHEN** apply runs with an MCP targeting claude and a claude setting
- **THEN** `mcpServers.<name>` is written to `~/.claude.json` and the setting to
  `~/.claude/settings.json`, with pre-existing unmanaged keys in both files intact

### Requirement: OpenCode projection

The OpenCode adapter SHALL project MCP servers into `opencode.jsonc`
(`mcp.<name>` with `type:"local"`, `command`, `enabled`, and `environment` when
env is set), settings as top-level keys, plugins appended to the `plugin` array,
and owned skills as symlinks under `~/.config/opencode/skills/`. JSONC input SHALL
be normalized before editing; loss of inline comments in rewritten regions is a
documented caveat.

#### Scenario: MCP projected with local shape and plugin appended
- **WHEN** apply runs with an MCP targeting opencode and an opencode plugin
- **THEN** `mcp.<name>.type` is `local` with the command, and the plugin is
  appended to the existing `plugin` array without duplicating existing entries

#### Scenario: Existing JSONC keys preserved
- **WHEN** `opencode.jsonc` has an unmanaged key and a comment
- **THEN** the unmanaged key survives after apply

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from `content/skills/<name>` into each
tool's skills directory. If the target already exists and is not homonto's link,
the adapter SHALL report a conflict and SHALL NOT clobber it.

#### Scenario: Idempotent link creation
- **WHEN** a skill symlink does not yet exist
- **THEN** apply creates it, and a second apply reports no change for that link

#### Scenario: Conflict is reported, not clobbered
- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched
```
