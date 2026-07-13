# Verification Report — onto-semantic-gates

**Date:** 2026-07-13 · ROADMAP N2 (last "Now" item) · Comet tweak · Result: PASS

- **onto close** now gates on close-phase evidence (workflow-aware): full requires
  verify.result==pass + close.merged + guides resolved; fix/tweak require
  verify.result==pass + close.merged (guides not required). `closeEvidenceGate` +
  `ontostate.GuidesResolved`. `7aec332`.
- **onto advance** gates on phase evidence: leaving verify requires verify.result==pass;
  entering build requires isolation set. `c1c4b7a`.
- Existing tests updated to provide the new evidence (seedClose, N7 conformance happy-path,
  one advance test) — gates not weakened.

## Evidence
`go test ./internal/ontocli/... ./internal/ontostate/... -race` → 135 passed; vet clean;
build OK; `openspec validate --all` 16/16.

## Scope note
Core B1 binary evidence-gating shipped. Deferred (separable N2 follow-ups, recorded in the
proposal): comet verification-scale-by-risk + non-waivable classes + skeptic/reviewer subagents
(F11/F12); skill-plane recovery/atomic bookkeeping (F16/F17); full dep resolver w/ cycles (F10).

## Milestone
Completes the "Now" horizon (gate A + gate B). All RC blockers already cleared.
