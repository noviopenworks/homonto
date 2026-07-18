# to reference — commands and behavior

`to` is the minimal coding framework for LLMs: three phases — **plan → do →
done** — a bookkeeper binary, and the `builtin:to` skills that carry the
process. This is the command surface; the design rationale is in
[to-framework-design.md](../to-framework-design.md), and the workflow prose
lives in the `to-*` skills homonto installs.

onto and `to` are an **exclusive choice** per repository: declaring both
`[frameworks.onto]` and `[frameworks.to]` in one `homonto.toml` fails at load.
Pick onto for evidence-gated enterprise changes, `to` for simple development.

## Install and enable

```bash
go install github.com/noviopenworks/homonto/cmd/to@latest
to version
```

The mutating commands (`init`, `new`, `phase`, `done`, `abandon`) require the
framework to be **declared and applied through homonto first** — this is how
the skills land in your tools:

```toml
[frameworks.to]
source = "builtin:to"
scope = "project"
# plus the [models.<tool>.*] routes — see the configuration reference
```

Then `homonto apply`. The read-only commands (`status`, `handoff`, `doctor`,
`version`) run without any of this — they never read `homonto.toml` and never
write.

## Layout

Each change is a directory `docs/tasks/<name>/` holding `to-state.yaml`
(written **only** by the binary) and `plan.md` (written by the agent during
plan). Finished changes move to `docs/tasks/archive/<date>-<name>/`; the date
prefix frees the name for reuse, and a same-day reuse gets a numeric suffix.
`to` is **git-blind**: it never inspects branches, worktrees, or dirt.

## Commands

Every command supports `--json` and `--dir <root>`.

| Command | What it does |
|---|---|
| `to init` | Scaffold `docs/tasks/` + `docs/tasks/archive/` (gated; never overwrites). |
| `to new <name>` | Create a change at phase `plan` with an empty `plan.md` (gated). Only an *active* change blocks a name. |
| `to phase <name>` | The one forward transition: `plan → do` (gated). Finishing is `to done`; there is no other advance. |
| `to done <name> --verified [--evidence "<text>"]` | Mark done and archive (gated). `--verified` is **required but self-asserted** — the binary records a checkbox, it observes nothing. `--evidence` records what was asserted, verbatim and unchecked, so a real verification is distinguishable in the archive. Requires phase `do`. |
| `to abandon <name>` | Terminal exit without done; archives (gated). |
| `to status` | Active changes and their phases. Read-only, config-independent. |
| `to handoff <name>` | Compact recovery pack: identity, phase, next command, and a plan excerpt (head + every unchecked `- [ ]` task) for resuming after a context compaction. Read-only, config-independent. |
| `to doctor [--quiet]` | Workspace health: invalid state files, wedged terminal-but-active changes (an interrupted archive — re-run the finishing command to converge), missing `plan.md`, a `do`-phase plan with no task checkboxes, non-terminal archive entries, and binary↔framework version skew. `--quiet` prints nothing and signals via exit code only — the hook primitive. Read-only, config-independent. |
| `to version` | The release-stamped version. |

## Crash safety

`done` and `abandon` write the terminal state, then move the directory into
the archive. If that is interrupted, the change is left terminal-but-active:
`to doctor` reports it, and **re-running the same finishing command completes
the archive** (`to done <name> --verified` / `to abandon <name>`). Mutating
commands take a workspace lock (`docs/tasks/.to.lock`), so two concurrent
sessions fail fast instead of interleaving writes; a lock left by a killed
process names its pid and is removed by hand.

## What `to` deliberately does not do

No evidence gates (the `--verified` checkbox is an assertion, not a
guarantee — the `to-done` skill is where verification rigor lives), no spec
deltas, no dependency graph, no git awareness, no parallel subagents, and no
escalation path to onto. If a change needs those, the repo needs onto.
