# Verification Report — onto-skills-shell-out

**Date:** 2026-07-13
**Change:** `onto-skills-shell-out` (ROADMAP N1, change B of A+B; Gate A onto Truth)
**Workflow:** Comet full · `verify_mode: full` · executed under /goal autonomous mode

## Result: PASS

## Full-verification checklist

| # | Check | Result |
|---|-------|--------|
| 1 | All `tasks.md` tasks `[x]` | PASS — 0 unchecked (9 groups) |
| 2 | Implementation matches change `design.md` | PASS — Layer 1 (binary extensions) + Layer 2 (8 skills shell out) + observational drop + guides field, as designed |
| 3 | Implementation matches Design Doc | PASS — field→command map realized; `guides` shape `pending|updated|waived:<reason>`; no schema_version bump |
| 4 | Capability spec scenarios pass | PASS — onto-binary delta scenarios (new --workflow, base-ref/deps setters, guides shape) covered by tests |
| 5 | `proposal.md` goals satisfied | PASS — every onto skill state write is a binary invocation; markdown-only/no-CLI copy deleted |
| 6 | Delta spec ↔ design doc consistency | PASS — no drift |
| 7 | Design docs locatable | PASS — `docs/superpowers/specs/2026-07-13-onto-skills-shell-out-design.md`, plan alongside |

## Enforcement evidence

- `scripts/onto-skills-shell-out-check.sh` (grep gate, wired into `scripts/gate.sh`):
  **PASS** — no `onto*` SKILL.md contains a direct state-file write instruction
  or the markdown-only/no-CLI copy. Proven to FAIL on the pre-rewrite skills
  (13 findings) and PASS after.
- `go test ./internal/ontostate/... ./internal/ontocli/... -race` → **121 passed**
- `go vet ./...` clean · `go build ./...` success · `openspec validate --all` 16/16

## Code review (standard)

No CRITICAL. One LOW finding **fixed** (`cf8668b`): a `waived:` guides value must
carry a non-empty reason (was accepting a bare `waived:`). All other invariants
(B1, write-nothing-on-failure, deps replace-not-append, legacy round-trip with
`Guides=""`) confirmed clean.

## Known deferrals (recorded, N2/N7)

- `onto new` writes `phase: open` always; preset (fix/tweak) working phase is
  derived from files, not written — workflow-aware transition *rules* are N2.
- The `abandoned:` field has no binary command; kept as a documented manual note
  in `onto/SKILL.md` (gate-excepted); `onto abandon` deferred to N2.
- Observational metrics dropped from the skills (never gated); the binary still
  carries the (now-empty) `Observed` fields — no schema change.
- Full-lifecycle onto skill dry-run belongs to N7 (the onto E2E suite).

## Sequencing

Canonical `openspec/specs/onto-binary/spec.md` is corrected by the archive
delta→main sync; archive runs before the branch lands on `main`.
