# Verify with adversarial fresh-context skeptic agents

- **Status:** Proposed
- **Date:** 2026-07-04
- **Change:** polish-onto-framework

## Context

Self-verification shares every blind spot with the implementation that
produced it. During onto v1, two fresh-context dry-run agents found 11
real defects (derivation-table direction, gate-skipping, YAML validity,
under-specified recovery) that in-session review had missed — evidence
that independent context, not more effort, is what catches this class of
error.

## Decision

We will make adversarial verification part of the verify phase: in full
mode, two parallel fresh-context skeptics — conformance (refute each
scenario claim) and robustness (edge cases, drift/recovery paths) — whose
findings are triaged into verification.md; light mode uses one optional
skeptic with skips recorded. Skeptics are prompted to refute, never to
approve. Absent subagent capability, the skipped pass is a recorded
deviation.

## Consequences

- Verification cost rises (two subagent runs) in exchange for catching
  plausible-but-wrong claims before close.
- Verification claims become falsifiable by construction: a refuted claim
  fails its scenario and routes through the existing fix-vs-accept gate.
- `verify_rounds` in metrics makes repeated failed rounds visible.
