# Design Notes: polish-onto-framework

Incremental checkpoint (compaction recovery). Unconfirmed items marked
*pending*.

## Confirmed (clarification, 2026-07-04)

- 7 axes: orchestration, templates, checkpoints, close lint, deps, ship
  handoff, metrics. One change (no split — same files).
- graphify index built: 353 nodes / 609 edges / 22 communities; grounding
  queries available via `graphify query`.
- Directive: run to completion (recorded in state.yaml decisions.directive).

## Graph-grounded observations

- Graph links "Drift Detection via state.json" ↔ "Phase Derivation and
  Cross-Check" (semantically_similar_to, INFERRED 0.85) — product and
  workflow share the reconciliation idiom; align vocabulary.
- onto Phase Contracts (C9) and onto Workflow Core (C11) communities are
  separate from v1 Design Decisions (C2) — the polish touches C9/C11 files
  only; no Go-code communities involved.

## Approach: CONFIRMED B (user gate, 2026-07-04)

Reference-file architecture — lean SKILL.md + bundled `references/`
templates/protocols. A and C rejected (context cost / binary dependency).
design.md written with Status: Confirmed; delta spec + 2 ADR drafts in
workspace.

## Confirmed decisions (were draft; now in design.md)

- Templates live in the phase skill that creates the artifact
  (onto-open/references/{proposal,tasks,state}.md, onto-design/references/
  {design,adr,delta-spec}.md, onto-build/references/plan.md,
  onto-verify/references/verification.md).
- Subagent build protocol: coordinator main session; per-task fresh
  implementer agent (task + files + verification + conventions + commit);
  reviewer agent for risky tasks; file-based checkoffs mandatory.
- Adversarial verify (full mode): 2 fresh skeptics — conformance (vs
  spec/design) + robustness (edge/drift) — instructed to REFUTE; findings
  triaged CRITICAL→fix, else deviations. Light mode: optional 1 skeptic.
- notes.md: this file's own pattern — updated each clarification/decision
  round in open/design; skills read it on entry; archived with the change.
- Close lint (agent-run, no scripts): delta format (SHALL first line,
  scenario GIVEN/WHEN/THEN, only ADDED/MODIFIED/REMOVED/RENAMED sections),
  RENAMED merge semantics added to specs README, post-merge no-delta-heading
  check, state.yaml enum validity, dangling-reference audit.
- deps: `deps: [<change>...]` in state.yaml; dispatcher lists blocked
  status, warns on resuming a change with unarchived deps; worktree
  guidance for parallel actives.
- Ship handoff: close offers a ready PR-body block (proposal why/what +
  verification summary + evidence pointers); saved as archive `ship.md`
  when accepted.
- Metrics: `metrics.phases.<phase>: <date>` stamped at each phase exit;
  close adds tasks_total, verify_rounds, upgraded (bool).
