# The onto workflow

**onto** is a spec-driven development workflow that homonto ships as a bundled
framework. It has two halves that work together:

- the **`onto` binary** (built from `cmd/onto/`, installed beside `homonto`) —
  the deterministic operator that creates change workspaces, gates phase
  transitions, merges spec deltas, and archives completed changes; and
- the **`onto-*` skills** (materialized from the builtin catalog by
  `homonto apply`) — the agent-facing process prose that drives the work inside
  each phase.

The binary owns the *state and the gates*; the skills own the *work*. A change
moves through five phases in a fixed order:

```
open → design → build → verify → close
```

`close` is terminal (the change is then archived by `onto close`). Each change
tracks its phase and gate evidence in an `onto-state.yaml` file inside its
workspace directory — always written through the binary, never by hand.

This guide covers the concepts. The precise command surface and every gate is
in the [onto reference](onto-reference.md); making the gates non-skippable at
the tool boundary is in [enforcement](enforcement.md).

## Install and enable

`onto` is version-stamped and installs alongside `homonto`:

```bash
go install github.com/noviopenworks/homonto/cmd/onto@latest
onto version            # prints: onto <version>
```

The mutating commands (`init`, `new`, `set`, `advance`, `close`, `abandon`,
`merge-deltas`) require the onto framework to be **declared and applied through
homonto first** — this is how the skills land in your tools:

```toml
[frameworks.onto]
source = "builtin:onto"
scope = "project"
# plus all three [models.<tool>.*] routes — see the configuration reference
```

Then `homonto apply`. The read-only commands (`status`, `state`, `gate`,
`scale`, `graph`, `handoff`, `doctor`, `version`) run without any of this —
they never read `homonto.toml` and never write.

`homonto apply` also installs the framework's **slash commands** into each
tool: `/onto` (the dispatcher — it derives the active change's real phase and
routes automatically), plus `/onto-open`, `/onto-design`, `/onto-build`,
`/onto-verify`, `/onto-close`, `/onto-fix`, `/onto-tweak`, and
`/onto-no-slop`. Each command loads the matching `onto-*` skill; every state
change still goes through the binary.

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

## Phase walkthrough

The `onto-*` skills carry the process discipline inside each phase; the binary
gates the transitions between them.

- **open** — clarify the requirement, decide whether the work should split into
  several changes, and create the workspace (`onto new`).
- **design** — ground-truth exploration, 2–3 candidate approaches, user
  confirmation, then `design.md`, ADR drafts (unnumbered, `Status: Proposed`),
  delta specs with testable scenarios, and the task list derived from the
  confirmed design. No implementation code in this phase.
- **build** — `plan.md` of bite-sized verified tasks, one commit per task,
  root-cause-first debugging on any failure. Entering build requires an
  isolation choice (`branch` or `worktree`), so build work is never committed
  unisolated.
- **verify** — scale-appropriate check of every delta-spec scenario with fresh
  command output as evidence, recorded in `verification.md`. `onto scale`
  derives the appropriate verification level from the measured diff.
- **close** — `onto merge-deltas` merges the change's delta specs into
  `docs/specs/` deterministically, then `onto close` archives the workspace
  once all evidence gates pass; number and accept ADRs into `docs/adr/`, and
  update the affected guides.

Two **presets** run a reduced path for small work: `onto new --workflow fix`
(an existing-behavior bug) and `--workflow tweak` (copy/config/docs-scale
change) go `open-lite → build → verify → close`, skipping design — and upgrade
to the full path when scope grows. `onto abandon` is the unsuccessful terminal
state for work that is stopped rather than completed.

## Specialist subagents

`homonto apply` installs the framework's subagents, which the onto skills
delegate to (don't also declare them in a top-level `[subagents.*]` table —
the names collide):

- **`onto-explorer`** — read-only; reads across many files to answer "how
  does X work / where does behavior live", returning conclusions, not dumps.
  Used for grounding in open/design. Runs on the `trivial` model route.
- **`onto-reviewer`** — read-only; reviews a diff for correctness, security,
  contract, and clarity, ranked by severity. Used per task in build and across
  the diff in verify. Runs on the `architectural` route.
- **`onto-implementer`** — edit-capable executor on the `coding` route.

All declare their capabilities once in a tool-neutral `homonto:` frontmatter
block, rendered into Claude's `tools:` allowlist and OpenCode's `permission:`
map (see [subagents](subagents.md)). Parallelization and dialogs work in both
tools: the build phase fans out independent tasks' investigation/review
concurrently, and gate decisions are asked through an interactive dialog
(`onto gate --json` supplies the structured decision; the skill renders it).
The orchestrator — your main session — still owns every edit and commit.

## Surviving context loss

Long agent sessions get compacted. `onto handoff <change>` emits a compact
recovery context pack — identity, phase, pending gate, artifact excerpts plus
a content hash — and `--write` persists it under the workspace, so a fresh
session resumes without re-deriving state. `onto set build-pause plan-ready`
records a first-class pause at the plan-ready gate for the same reason.

## Recommended tooling

The onto skills recommend two tools; when either is missing they warn and
proceed — a degraded session still works:

- **rtk** — a token-optimized CLI proxy; workflow shell operations go through
  it when installed. Missing rtk means higher token cost, never a stop.
- **graphify** (https://graphify.net) — codebase understanding; the open and
  design phases ground claims in graphify/codegraph queries when available,
  falling back to direct file reading otherwise.

> homonto's own repository is developed with **Comet**, not onto — see
> [comet-workflow.md](comet-workflow.md) and
> [`docs/personas.md`](../personas.md). onto is a shipped product framework;
> this guide documents it for projects that adopt it.
