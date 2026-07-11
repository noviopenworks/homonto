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
