# Design: polish-onto-framework

Status: Confirmed
Confirmed: 2026-07-04 (approach B — reference-file architecture)

## Summary

onto v2 keeps the eight lean SKILL.md files as *process* and moves *payload*
— canonical artifact templates and detailed protocols — into `references/`
files bundled inside each skill directory (progressive disclosure: loaded
only when the phase needs them, travelling with the same symlinks to every
repo). On that foundation the seven axes land: a defined subagent build
protocol, adversarial multi-agent verification, notes.md checkpoints,
close-phase lint, `deps:` coordination, ship handoff, and archive metrics.

Rejected: A (everything inline — doubles every skill's context cost on
every invocation); C (homonto lint subcommand — reintroduces the binary
dependency ADR 0005 removed).

## Goals / Non-Goals

**Goals:** the seven user-selected axes, dogfooded in this very change
where possible (notes.md already in use). **Non-goals:** no Go source
changes; no new top-level skills (eight remain eight); no change to gate
semantics established in v1 (gates stay sacred; pre-authorization rules
unchanged).

## Architecture

### Reference-file layout (approach B)

```
content/skills/
├── onto/SKILL.md                  # + references/state-yaml.md (schema+template)
├── onto-open/SKILL.md             # + references/{proposal,tasks,notes}.md
├── onto-design/SKILL.md           # + references/{design,adr-draft,delta-spec}.md
├── onto-build/SKILL.md            # + references/{plan,subagent-protocol}.md
├── onto-verify/SKILL.md           # + references/{verification,adversarial}.md
├── onto-close/SKILL.md            # + references/{lint-checklist,ship-handoff}.md
├── onto-fix/SKILL.md              # (reuses open/build/verify/close references)
└── onto-tweak/SKILL.md            # (reuses open/build/verify/close references)
```

Rules: SKILL.md instructs *when* to read a reference ("create proposal.md
from `references/proposal.md`"); references are canonical — an artifact that
deviates from its template structure is a lint finding at close. Each
template is a complete fenced skeleton plus per-field rules. Contracts in
`docs/` (changes/adr/specs READMEs) point at the templates instead of
duplicating them, keeping single-source-of-truth in the skill tree; the one
deliberate exception stays the phase-derivation table (dispatcher +
changes README, byte-identical, because the dispatcher must work standalone).

### state.yaml additions

```yaml
deps: []                   # change names that must archive before this one builds
metrics:                   # stamped at phase exits; finalized at close
  phases: {open: 2026-07-04, design: …, build: …, verify: …, close: …}
  tasks_total: 0
  verify_rounds: 0
  upgraded: false          # preset→full upgrade happened
```

Rebuild rules (README + dispatcher): `deps` from the proposal's
`Depends-on:` line if present else `[]`; `metrics.phases` from the git
dates of the commits that advanced each phase, else omitted (metrics are
best-effort — never block on reconstruction).

### Subagent build protocol (`execution: subagent`)

Coordinator/worker: the main session NEVER implements; it dispatches one
fresh-context implementer agent per task with: the task text + exact file
paths, the confirmed design section, conventions (commit-per-task, message
style), the task's stated verification, and the systematic-debugging rule.
The implementer implements, runs verification, checks the task off in
tasks.md AND plan.md, commits, and returns a diff summary + verification
output. The coordinator verifies the commit exists and the checkoff
happened (files, not chat), then dispatches the next task. A reviewer agent
is dispatched after any task the plan marks `risk: high` (and always for
the final task), prompted to find faults, not to approve. Choose
`execution: subagent` when tasks are numerous/independent or main-session
context is precious; `direct` remains right for small serial changes.
Details: `onto-build/references/subagent-protocol.md`.

### Adversarial verification

After the evidence table is drafted (self-verification), `verify.mode:
full` REQUIRES dispatching two fresh-context skeptic agents in parallel,
each with the delta specs + design + repo access and the instruction to
REFUTE, not confirm: a **conformance skeptic** (does the implementation
actually satisfy each scenario? try to break the claims) and a
**robustness skeptic** (edge cases, drift/recovery paths, gaps the
scenarios don't cover). Their findings are triaged into the report:
refuted claim → verification fails that scenario; new defect → CRITICAL
(fix) or deviation (gate). `verify_rounds` increments per round. Light
mode: one skeptic, optional (skip recorded in the report). Precedent: the
v1 dry-run agents found 11 real defects self-review missed.
Details: `onto-verify/references/adversarial.md`.

### notes.md checkpoint

`docs/changes/<name>/notes.md`, created at open, updated after every
clarification round, approach iteration, and design decision — confirmed
facts vs *pending* items explicitly separated. onto-open and onto-design
MUST update it before ending any turn that produced new decisions; all
skills read it at entry when present. It is the compaction-recovery
complement to the derivation table: derivation recovers *where* you are,
notes.md recovers *why*. Archived with the change.
Template: `onto-open/references/notes.md`.

### Close-phase lint (agent-run, no scripts)

Staged per `onto-close/references/lint-checklist.md` (§1–2 pre-merge, §3
post-merge, §4 pre-archive):

1. Delta spec format: sections only ADDED/MODIFIED/REMOVED/RENAMED; every
   ADDED/MODIFIED requirement's first non-empty line contains SHALL or
   MUST; every such requirement has ≥1 `#### Scenario:` with
   GIVEN/WHEN/THEN bullets.
2. RENAMED semantics (new in specs README): `## RENAMED Requirements` with
   `- FROM: <name>` / `  TO: <name>` pairs; merge renames the requirement
   heading in the living spec, preserving its body unless a MODIFIED block
   also targets the new name.
3. Post-merge: living specs contain no delta-only headings; scenario
   structure intact.
4. Workspace: state.yaml parses with valid enums; verification.md has a
   `Result:` line; ADR drafts carry Status/Date/Change fields.
5. Dangling-reference audit over live docs.

Lint findings block the archive step exactly like the guides obligation.

### deps coordination

Dispatcher discovery lists each active change with `deps` status; resuming
a change whose deps are not all archived → warn and require explicit user
choice (proceed anyway / switch to the dep / stop). Multiple active
changes: recommend worktree-per-change (`git worktree`), one active change
per worktree. No cross-change file locking — coupled changes should be one
change (split-preflight rule already says so).

### Ship handoff

onto ends at a closed change; ship stays outside — but close now *prepares*
the handoff: after archive, offer a ready PR body (proposal why/what +
verification summary + evidence pointers + archive path). If accepted,
write it to `docs/changes/archive/YYYY-MM-DD-<name>/ship.md` and name the
PR skills as the next step. Contract: `onto-close/references/ship-handoff.md`.

### Metrics

Each phase's exit checklist stamps `metrics.phases.<phase>: <date>`; close
finalizes tasks_total (checked tasks), verify_rounds, upgraded. Metrics are
observational only — never a gate, never blocking.

## Key decisions

1. **Reference-file skill architecture** (→
   `adr/reference-file-skill-architecture.md`): payload in bundled
   references, process in lean SKILL.md — progressive disclosure over
   inline bulk (A) or a lint subcommand (C).
2. **Adversarial multi-agent verification** (→
   `adr/adversarial-multi-agent-verification.md`): fresh-context skeptics
   prompted to refute, never approve — independence over redundancy,
   grounded in the v1 dry-run precedent (11 defects self-review missed).

## Error handling

- Missing reference file (skill symlinked but references pruned): the
  SKILL.md carries a one-line fallback ("if references/ is missing,
  reconstruct from docs/changes/README.md pointers and note the gap") —
  degraded, never halted.
- Skeptic agents unavailable (no dispatch capability): record
  "adversarial pass skipped: no subagent capability" in the report's
  Adversarial section (no acceptor needed); verification may still pass
  with the skip recorded.
- Lint findings at close: same blocking flow as guides obligation.

## Testing strategy

1. Template conformance: this change's own artifacts must match the new
   templates (self-application).
2. Fresh-context dry-run A: full lifecycle using templates + notes.md +
   subagent protocol simulation.
3. Fresh-context dry-run B: adversarial verify + close lint (feed it a
   deliberately malformed delta spec — must catch SHALL/scenario/RENAMED
   errors) + deps warning + metrics stamping.
4. Self-containment grep (openspec/comet/superpowers-free) extended over
   references/; derivation-table byte-identity check.
5. `go test ./...` stays green (no Go changes).

## Grounding

- C9 (onto Phase Contracts) + C11 (onto Workflow Core) are the touched
  communities; no Go community membership → no product-code risk.
- The graph's `Drift Detection ↔ Phase Derivation` similarity edge
  motivated the aligned vocabulary used in the state additions ("stated
  state vs verifiable reality" reconciliation, best-effort rebuild).
