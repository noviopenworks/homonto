# Notes: polish-onto-framework

Incremental checkpoint (compaction recovery). Unconfirmed items are
marked *pending*.

## Confirmed

- 2026-07-04 clarification gate: 7 axes — orchestration, templates,
  checkpoints, close lint, deps, ship handoff, metrics ("Artifact
  templates, Context-loss checkpoints, Close-phase validation,
  Multi-agent orchestration" + "Ship handoff, Metrics in archive,
  Parallel-change coordination"). One change, no split (same files).
- 2026-07-04 graphify: user chose "Yes — build graphify index"; built
  (353 nodes / 609 edges / 22 communities).
- 2026-07-04 directive (recorded in state.yaml decisions.directive):
  "run to completion" — pre-answers plan-ready and close gates only.
- 2026-07-04 artifact-review gate: "Approve — name ok" for
  polish-onto-framework.
- 2026-07-04 approach gate: "B: Reference files" confirmed (lean SKILL.md
  + bundled references/, progressive disclosure). A and C rejected
  (context cost / binary dependency).
- 2026-07-04 verify-fail gate (round 1): "Fix all, round 2" — all 16
  triaged findings fixed and committed.
- 2026-07-04 verify-fail gate (round 2): "Fix all + round 3 light" — 14
  residual findings (4 substantive: revision-path completion, gate→
  boundary mapping, downgrade guard, DEFERRED resolution; 10
  harmonizations) all fixed.
- 2026-07-04 plan-ready gate: pre-authorized by the recorded directive;
  config branch/direct/direct (recorded here per the build→verify gate
  cap).

## Pending

- Verification round 3 (light, single focused skeptic on the round-2
  fixes) → then close.

## Grounding

- graphify index over the repo; key edge: Drift Detection ↔ Phase
  Derivation (semantically_similar_to, 0.85) — informed the aligned
  reconciliation vocabulary in state-yaml.md.
- Direct reads of all 8 SKILL.md + 13 references during build and both
  dry-run agents' walks.

## Approaches

- A: everything inline in SKILL.md — rejected (context cost per dispatch).
- B: reference-file architecture — **CONFIRMED 2026-07-04**.
- C: homonto lint subcommand — rejected (binary dependency, ADR 0005).
