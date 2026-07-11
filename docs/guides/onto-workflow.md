# The onto Workflow

**onto** is a spec-driven development workflow that Homonto ships as a bundled
framework. It has two halves that work together:

- the **`onto` binary** (built from `cmd/onto/`, installed beside `homonto`) —
  the deterministic operator that creates change workspaces, gates phase
  transitions, and archives completed changes; and
- the **`onto-*` skills** (materialized from the builtin catalog by `homonto
  apply`) — the agent-facing process prose that drives the work inside each
  phase.

The binary owns the *state and the gates*; the skills own the *work*. A change
moves through five phases in a fixed order:

```
open → design → build → verify → close
```

`close` is terminal. Each change tracks its phase and gate fields in an
`onto-state.yaml` file inside its workspace directory.

## Install and enable

`onto` is version-stamped and installs alongside `homonto`:

```bash
go install github.com/noviopenworks/homonto/cmd/onto@latest
onto version            # prints: onto <version>
```

The mutating commands (`init`, `new`, `advance`, `close`) require the onto
framework to be **declared and applied through Homonto first** — this is how
the skills land in your tools. In `homonto.toml`:

```toml
[frameworks.onto]
source = "builtin:onto"
```

Then `homonto apply`. The read-only commands (`status`, `doctor`, `version`)
run without any of this — they never read `homonto.toml` and never write.

## The layout

`onto init` scaffolds four directories under the workspace root (idempotently —
existing content is never overwritten):

```
docs/
├── changes/                # change workspaces + archive
│   ├── <name>/             # active change (onto-state.yaml, proposal, …)
│   └── archive/YYYY-MM-DD-<name>/
├── specs/                  # living capability specs
├── adr/                    # accepted / superseded decisions
└── guides/                 # user-facing docs
```

## Commands

| Command | Phase gate | What it does |
|---|---|---|
| `onto init` | framework-install | Scaffold the `docs/{changes,specs,adr,guides}/` layout. Idempotent; reports created vs. skipped paths. |
| `onto new <name>` | framework-install | Create `docs/changes/<name>/` with an `onto-state.yaml` (phase `open`), `proposal.md`, and `tasks.md`. Refuses to clobber an existing change; validates the name is kebab-case with no path traversal. |
| `onto status` | none (read-only) | Report each discovered change's derived phase and skeleton validity. Config-independent; writes nothing. |
| `onto advance <change>` | framework-install + artifact/tasks gates | Move a change one step along `open→design→build→verify→close`. |
| `onto close <change>` | framework-install + deps + clean worktree | Archive a completed change to `docs/changes/archive/<date>-<change>/`. |
| `onto doctor [--dir <root>]` | none (read-only) | Diagnose workspace health across docs layout, active-change state, phase/artifact match, dependency resolution, and archive layout. Exits non-zero on any finding. |
| `onto version` | none | Print the release-stamped version. |

## The gates

`onto advance` only leaves a phase once that phase's deliverables exist. The
required artifacts accumulate as a change advances:

| Leaving phase | Requires |
|---|---|
| `open` | `proposal.md`, `tasks.md` |
| `design` | + `design.md` |
| `build` | + `plan.md` **and every `tasks.md` checkbox checked** (no unchecked `- [ ]`) |
| `verify` | + `verification.md` |

A missing deliverable makes `advance` exit non-zero and leaves the recorded
phase unchanged. Advancing a change already at `close` is an error.

**Dirty-worktree handling.** `advance` checks `git status --porcelain`. For a
normal transition a dirty worktree is a *warning* but still allowed; for the
release-critical `verify → close` transition it **blocks** the advance. `onto
close` likewise refuses to archive unless the worktree is clean, the change is
at phase `close`, and every dependency listed in its `onto-state.yaml` is
resolved (an archived `docs/changes/archive/*-<dep>` exists).

## Phase walkthrough

The `onto-*` skills carry the process discipline inside each phase; the binary
gates the transitions between them.

- **open** — clarify the requirement, decide whether the work should split into
  several changes, and create the workspace (`onto new`).
- **design** — ground-truth exploration, 2–3 candidate approaches, user
  confirmation, then `design.md`, ADR drafts (unnumbered, `Status: Proposed`),
  and delta specs with testable scenarios. No implementation code in this phase.
- **build** — `plan.md` of bite-sized verified tasks, one commit per task,
  root-cause-first debugging on any failure.
- **verify** — scale-appropriate check of every delta-spec scenario with fresh
  command output as evidence, recorded in `verification.md`.
- **close** — `onto close` archives the workspace once all gates pass; merge
  delta specs into `docs/specs/`, number and accept ADRs into `docs/adr/`, and
  update the affected guides.

## Recommended tooling

The onto skills recommend two tools; when either is missing they warn and
proceed — a degraded session still works:

- **rtk** — a token-optimized CLI proxy; workflow shell operations go through it
  when installed. Missing rtk means higher token cost, never a stop.
- **graphify** (https://graphify.net) — codebase understanding; the open and
  design phases ground claims in graphify/codegraph queries when available,
  falling back to direct file reading otherwise.

> Homonto's own repository is developed with **Comet**, not onto — see
> [comet-workflow.md](comet-workflow.md). onto is a shipped product framework;
> this guide documents it for projects that adopt it.
