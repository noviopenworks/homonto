## Why

v2 added the `[agents.<name>]` model, `agents list`, and `agents add` (with the
`.homonto/agents-lock.json` lockfile of what's installed). The next lifecycle
piece is *health*: comparing what's declared (config) against what's installed
(lockfile + disk) so a user knows an agent is missing, orphaned, or drifted
before `update`/`migrate` land. This change adds `homonto agents doctor` — a
read-only diagnostic, the peer of `homonto doctor`/`onto doctor` for the agent
lifecycle. It also unblocks later increments (`update`/`migrate` act on the drift
`doctor` reports).

## What Changes

- Add `homonto agents doctor`: read-only, loads the config (declared agents) and
  `.homonto/agents-lock.json` (installed), and reports each problem as a finding:
  - **declared but not installed** — a declared agent with no lockfile record
    (run `agents add`);
  - **orphaned** — a lockfile-recorded agent no longer declared in config;
  - **source drifted** — a `local:` agent whose `homonto/agents/<x>.md` content
    hash no longer matches the recorded install hash (re-run `agents add`), or
    whose source file is now missing;
  - **target not installed** — a target the agent declares that has no lockfile
    install entry (e.g. a newly added target);
  - **target no longer declared** — an installed target the agent no longer
    targets;
  - **missing on disk** — a recorded install path that no longer exists;
  - **modified on disk** — a `copy`-mode install whose on-disk content hash no
    longer matches the recorded hash.
  On a healthy workspace it prints `healthy` and exits 0; with findings it prints
  each and exits non-zero (CI/scriptable, like `onto doctor`). It writes nothing.
- Register `doctor` under `agentsCmd()`. `homonto agents` now has list / add /
  doctor.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents doctor`, a read-only health check
  reporting declared-vs-installed drift (missing/orphaned/source-drifted/
  target-mismatch/missing-on-disk/modified-on-disk), non-zero exit on findings.

## Impact

- `internal/cli/agents.go`: new `doctor` subcommand (`agentsDoctorCmd`).
- Reuses `internal/agentlock` (`Load`, `HashContent`) and `internal/subagentpath`
  (only for path context if needed — findings use recorded paths).
- Tests in `internal/cli`.
- No new dependency. Read-only; no projection/mutation. All prior behavior
  unchanged.
- Deferred: `update`/`pin`/`migrate` (which act on this drift), builtin/remote
  sources, three-way-merge.
