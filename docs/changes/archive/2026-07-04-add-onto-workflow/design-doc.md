---
comet_change: add-onto-workflow
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-04-add-onto-workflow
status: final
---

# onto Workflow — Technical Design

**Date:** 2026-07-04
**Status:** Confirmed (user, 2026-07-04)

## Summary

**onto** is a self-contained, markdown-only development workflow shipped as
homonto-owned content. It keeps Comet's rigor (phased lifecycle, blocking
decision points, TDD discipline, verification evidence, spec lifecycle) while
removing all external machinery: no `openspec` CLI, no bash guard/state
scripts. Artifacts live in a single `docs/` tree per project; phase state is a
tiny agent-managed YAML file that the dispatcher always cross-checks against
verifiable file state.

## Implementation Divergence (recorded at verify, 2026-07-04)

The non-goal "no Go code changes" was superseded mid-build with user
approval: dogfooding exposed two real product bugs (skills-only configs
never applied symlinks because links were absent from the plan ChangeSet;
relative content dirs produced dangling symlink targets). Both were fixed
TDD-style in `internal/link`, `internal/adapter/{claude,opencode}`, and
`internal/engine`, with a `tool-adapters` delta spec covering the new
behavior. proposal.md's Modified Capabilities section records the scope
amendment; the user affirmed the implemented state as authoritative.

## Goals / Non-Goals

**Goals:** self-contained workflow skills; ADR + living-spec + post-impl-guide
documentation model; comet-grade rigor; rtk + graphify hard-required; GitHub
skills as entry points; dogfooded via `homonto apply`; homonto repo migrated.

**Non-goals:** Go code changes; PR-creation/review automation inside phases;
editing global comet/openspec/superpowers skills; remote distribution.

## Architecture

### Skill set (8 skills, `content/skills/<name>/SKILL.md`)

| Skill | Role |
|---|---|
| `onto` | Dispatcher: tooling preflight, active-change discovery, phase derivation, routing, resume/recovery, entry-point contract |
| `onto-open` | Clarify → split preflight → `proposal.md` + `tasks.md` skeleton → review gate |
| `onto-design` | Brainstorming-grade design → `design.md` + ADR drafts + spec deltas → approach gate |
| `onto-build` | `plan.md` → plan-ready gate → execute tasks (TDD or direct) → commit per task |
| `onto-verify` | Scale check → evidence-based verification vs design/spec scenarios → `verification.md` → fail gate |
| `onto-close` | Merge spec deltas → number+accept ADRs → guides obligation → archive → final gate |
| `onto-fix` | Bug preset: open-lite → build (failing test first) → verify → close; upgrade rules |
| `onto-tweak` | Small-change preset: open-lite → lightweight build → light verify → close; upgrade rules |

Each sub-skill is independently loadable: it restates its entry check
(expected `phase`), so a cold session can start from any phase.

### Per-project artifact layout (the "layout contract")

```
docs/
├── adr/
│   ├── README.md                    # format + numbering rules
│   └── NNNN-<slug>.md               # accepted/superseded decisions only
├── specs/
│   ├── README.md                    # living-spec contract
│   └── <capability>.md              # SHALL requirements + Given/When/Then scenarios
├── changes/
│   ├── README.md                    # workspace + state.yaml contract
│   ├── <name>/                      # active change workspace
│   │   ├── state.yaml
│   │   ├── proposal.md              # why + what + capability impact
│   │   ├── design.md                # confirmed technical design (full workflow only)
│   │   ├── adr/<slug>.md            # ADR drafts, status: Proposed, unnumbered
│   │   ├── specs/<capability>.md    # delta: ADDED/MODIFIED/REMOVED requirements
│   │   ├── plan.md                  # implementation plan (full workflow)
│   │   ├── tasks.md                 # checklist, one commit per task
│   │   └── verification.md          # evidence-based verify report
│   └── archive/YYYY-MM-DD-<name>/   # closed changes, workspace moved verbatim
└── guides/
    └── <topic>.md                   # user-facing docs, written/updated at close
```

### state.yaml (agent-managed)

```yaml
change: add-foo            # must equal directory name
workflow: full             # full | fix | tweak
phase: build               # open | design | build | verify | close
created: 2026-07-04
base_ref: <git sha at open>
decisions:                 # null until chosen (build entry)
  isolation: branch        # branch | worktree
  execution: direct        # direct | subagent
  tdd: tdd                 # tdd | direct
verify:
  mode: null               # light | full (set at verify entry by scale rules)
  result: pending          # pending | pass | fail
guides: pending            # pending | updated | waived: <reason>
archived: false
```

Rules: the agent edits this file directly (no scripts). **It is a cache of
truth, not truth.** On every `/onto` dispatch the phase is re-derived from
artifact state and cross-checked; mismatch → correct state.yaml to match
files, tell the user, continue from real state.

Phase-derivation table (first match from bottom wins):

| Evidence | Real phase |
|---|---|
| `archived: true` or workspace under `archive/` | done |
| `verification.md` exists + `verify.result: pass` | close |
| all tasks checked in `tasks.md` | verify |
| `design.md` confirmed (or preset) + plan/tasks in progress | build |
| `proposal.md` + `tasks.md` exist, no confirmed design | design (full) / build (preset) |
| workspace exists, artifacts incomplete | open |

### Blocking decision points (kept from comet)

1. open: clarification-complete + artifact review
2. design: approach confirmation (before writing final design.md)
3. build: plan-ready pause + execution config (isolation/execution/tdd) —
   may be pre-authorized by an explicit user directive recorded in state.yaml
4. verify fail: fix vs accept-deviation
5. close: final confirmation before archive
6. preset upgrade triggers (fix/tweak → full)

A blanket user directive ("run to completion") may pre-answer 3 and 5; the
skill must record the directive verbatim under `decisions:` and still surface
summaries. 1, 2, 4 and 6 always require fresh user input unless the user
explicitly pre-answered that same question.

### Required tooling (hard requirement)

Dispatcher preflight, in order:
1. `rtk --version` succeeds → all subsequent shell commands go through rtk
   (per RTK.md hook conventions). Fail → halt: install instructions.
2. graphify available (skill present) → open/design phases must ground
   codebase understanding in `/graphify` queries (or existing
   `graphify-out/` / `.codegraph/` index). Fail → halt: install instructions.

### GitHub entry points (contract, not implementation)

- **resolve-issue** → entry to `/onto`: issue text seeds `onto-open`
  clarification (fix preset for bugs, full for features), worktree isolation.
- **continue-pr** → entry to `/onto`: PR feedback resumes the matching change
  (reopens build) or opens a `fix` change referencing the PR.
- PR creation/review stay in their own skills; onto ends at a verified,
  closed change on a branch.

### Design rigor (carried from Superpowers)

- design phase follows brainstorming discipline: explore ground truth first,
  question until clear, 2–3 approaches, confirm before writing.
- build follows writing-plans + TDD disciplines: bite-sized tasks with exact
  files/verification, failing test first when `tdd: tdd`, commit per task,
  systematic-debugging gate on any failure (root cause before fix).
- verify follows verification-before-completion: fresh evidence (commands +
  output) for every claim; report lists each requirement scenario → verdict.

## Key Decisions (→ ADR drafts)

1. **adopt-onto-workflow**: replace comet+openspec machinery with
   self-contained markdown skills + agent-managed state (this design).

Extracted from existing v1 decisions during migration (status Accepted,
sourced from v1-core design/README):
2. plan-confirm-apply pipeline with tool adapters
3. secrets referenced-never-stored (hash-only state)
4. owned content symlinked, never clobbered; surgical merge
5. atomic writes, state-written-last

## Migration Plan (this repo)

| Source | Destination |
|---|---|
| `openspec/specs/<cap>/spec.md` (5) | `docs/specs/<cap>.md` |
| `openspec/changes/archive/2026-07-03-homonto-v1-core/` | `docs/changes/archive/2026-07-03-homonto-v1-core/` |
| `docs/superpowers/specs/2026-07-03-homonto-v1-core-design.md` | same archive dir, `design-doc.md` |
| `docs/superpowers/plans/2026-07-03-homonto-v1-core.md` | same archive dir, `plan.md` |
| `docs/superpowers/reports/2026-07-03-homonto-v1-core-verify.md` | same archive dir, `verification.md` |
| `docs/superpowers/specs/2026-06-24-homonto-design.md` + `plans/2026-06-24-homonto.md` | `docs/changes/archive/2026-06-24-homonto/` |
| `docs/superpowers/specs/2026-07-03-homonto-roadmap.md` | `docs/roadmap.md` |
| decisions embedded in v1 design/README | `docs/adr/0001..0004` |
| `openspec/`, `docs/superpowers/` | removed at the end |

Bootstrap ordering: this change itself runs under comet. Build migrates
everything **except** the active change workspace. At comet-archive, the
archive script runs first (moves workspace, merges delta spec), then final
tasks move the merged spec + archived workspace into `docs/` and delete
`openspec/` + `docs/superpowers/`.

## Dogfood Wiring

- Create root `homonto.toml`: `[skills] own = ["onto", "onto-open",
  "onto-design", "onto-build", "onto-verify", "onto-close", "onto-fix",
  "onto-tweak"]`.
- `go build -o homonto . && ./homonto apply` → plan/confirm → symlinks
  `content/skills/onto*` → `~/.claude/skills/onto*` (user-global; adapter
  links into `$HOME/.claude/skills/<name>`, linker never clobbers).
- Proof: links exist, `homonto status` clean, skills resolve in a session.

## Error Handling

| Scenario | Handling |
|---|---|
| state.yaml missing/malformed | Rebuild from phase-derivation table; announce correction |
| state.yaml vs files conflict | Files win; correct state.yaml; announce |
| rtk/graphify missing | Halt with install instructions (hard requirement) |
| build/test failure | systematic-debugging gate; no fix before root cause |
| verify fail | blocking point: fix (→ build) vs accept deviation (recorded in verification.md) |
| upgrade trigger in preset | blocking point: confirm upgrade → backfill design phase |
| two active changes | dispatcher lists, asks which to resume |
| guides not updated at close | close cannot complete: update or record `waived: <reason>` |

## Testing Strategy

1. **Dry-run walkthroughs** (agent-simulated, in-repo scratch change):
   full lifecycle open→close; fix preset incl. upgrade trigger; tweak preset;
   drift-recovery (hand-corrupt state.yaml, dispatcher must self-correct).
2. **Self-containment check**: `grep -r` over `content/skills/onto*` proves
   zero references to `openspec` CLI, comet scripts, or `docs/superpowers/`.
3. **Dogfood check**: `homonto plan`/`apply`/`status` output captured;
   symlinks verified with `ls -l ~/.claude/skills/onto*`.
4. **Regression**: `go test ./...` stays green (no Go changes).
5. **Migration audit**: every source file accounted for (git mv preserved
   history where possible); no dangling references to old paths in README or
   skills.
