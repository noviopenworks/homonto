## Why

The v2 foundation added the `[agents.<name>]` model and read-only `homonto
agents list`. The next step in the agent lifecycle is *installing* a declared
agent: materializing it into the target tools and recording what was installed so
later increments (update/pin/doctor/migrate) have ground truth. This change adds
`homonto agents add` plus the agent lockfile — the first agent-lifecycle
mutation, and the ground truth the rest of v2 builds on. Scope is kept
self-contained: `local:` sources only (builtin/remote deferred), `copy` and
`link` modes, conflict-safe and idempotent.

## What Changes

- Add an agent lockfile at `.homonto/agents-lock.json` (a new `internal/agentlock`
  package: typed model + `Load`/`Save`, empty on absence). Per installed agent it
  records `source`, `version`, `mode`, `targets`, and per-target
  `{path, hash}` (sha256 of the installed content). The lockfile is separate from
  `state.json` (agent lifecycle needs its own installed-version ground truth).
- Add `homonto agents add <name>`: installs a declared agent.
  - Loads the config, finds `[agents.<name>]` (error if undeclared).
  - Supports `source = "local:<x>"` → resolves `homonto/agents/<x>.md` relative to
    the config dir (error if missing). `builtin:`/remote sources return a clear
    "not yet supported" error (deferred).
  - For each target in the agent's `TargetsOrAll()`: the destination is
    `<agent dir for tool>/<name>.md` (via `subagentpath.Dir`, user scope). `copy`
    mode writes the file content; `link` mode symlinks the source.
  - **Conflict-safe**: if a destination already exists and is NOT a homonto-managed
    install of this agent (absent from the lockfile), it REFUSES and installs
    nothing for that agent (all-or-nothing).
  - **Idempotent**: a target already installed with matching content/hash is a
    no-op; re-running reports "already up to date".
  - Updates the lockfile and prints what was installed/updated per target.
- This is additive; `agents list`, the `[subagents]` projection, and `plan`/
  `apply` are unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents add` (install a `local:` agent per its
  mode into target tools, conflict-safe and idempotent) and the
  `.homonto/agents-lock.json` lockfile recording installed agents.

## Impact

- New `internal/agentlock` package (lockfile model + Load/Save + hashing helper).
- `internal/cli/agents.go`: new `add` subcommand (`agentsAddCmd`).
- Reuses `internal/subagentpath.Dir`, `internal/fsutil.WriteAtomic`,
  `internal/link.Link`.
- Tests in `internal/agentlock` and `internal/cli`.
- No new dependency. Read-only `agents list` and all v1 behavior unchanged.
- Deferred to later increments: `builtin:`/remote sources; `update`/`pin`/
  `doctor`/`migrate`; three-way-merge/backup; a per-agent scope field;
  `[agents]`-vs-`[subagents]` reconciliation.
