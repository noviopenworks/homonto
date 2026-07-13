# Verification Report — fix-stale-canonical-specs

**Date:** 2026-07-13
**Change:** `fix-stale-canonical-specs` (ROADMAP N3 / finding F5)
**Workflow:** Comet tweak · `verify_mode: full` (3 delta capabilities, 12 tasks, 17 files)

## Result: PASS

## Full-verification checklist

| # | Check | Result |
|---|-------|--------|
| 1 | All `tasks.md` tasks `[x]` | PASS — `grep -c '^- \[ \]'` = 0 |
| 2 | Implementation matches change `design.md` decisions | PASS — agent-lifecycle retired, one surviving truth folded into config-model, coarse CI check added, exactly as designed |
| 3 | Superpowers Design Doc consistency | N/A — tweak preset has no `docs/superpowers/specs/` design doc; change-local `design.md` is the design of record |
| 4 | Capability spec scenarios match shipped source | PASS — see source-truth table below |
| 5 | `proposal.md` goals satisfied | PASS — all three goals (correct specs, CI guard, doc verification) met |
| 6 | Delta spec ↔ design doc consistency | PASS — deltas and `design.md` agree; no drift |
| 7 | Design docs locatable | N/A (tweak) — change-local artifacts present |

## Source-truth evidence (a spec-truth change must have true claims)

| Delta claim | Verified against |
|-------------|------------------|
| CLI registers only `version/init/import/plan/apply/status/doctor`, no `agents` | `internal/cli/root.go:20-28` |
| `[agents.<name>]` folds into a copy-mode, user-scope subagent at load; agents win over same-named subagents; `c.Agents=nil` | `internal/config/config.go:509-527` (`:518` fold, `:526` nil) |
| `agentlock`/`agentblob`/`internal/cli/agents.go` removed | absent from tree |
| "Three-way merge engine" requirement is orphaned (safe to remove) | `internal/merge` has zero non-test callers |
| README/guide already state the fold | `README.md:118`, `docs/guides/using-homonto.md:14` |

## Command evidence

- `openspec validate fix-stale-canonical-specs` → valid
- `openspec validate --all` → 16 passed, 0 failed
- `go build ./...` → success · `go vet ./...` → no issues
- `scripts/spec-command-check.sh`: **fails** on the current canonical specs
  (2 violations: `agent-lifecycle`, `cli-commands` name `homonto agents`, exit 1);
  **passes** on a preview of the post-sync tree (exit 0)

## Known sequencing gap (by design, not a defect)

The canonical `openspec/specs/*` corrections are applied by the delta→main sync
that **archive** performs — that sync *is* this change's deliverable. Until then,
`scripts/spec-command-check.sh` (now wired into `scripts/gate.sh`) is red on the
canonical tree. Therefore: **archive must run before this branch merges to main**,
so the branch that lands carries corrected specs and a green gate. Full `gate.sh`
is green post-archive. This gap is expected for a change that fixes the very specs
its new check guards.

## Archive-gate amendment (2026-07-13)

The first archive attempt aborted cleanly (no files changed): the `agent-lifecycle`
delta removed *all* requirements, and OpenSpec cannot rebuild an empty spec
("Spec must have at least one requirement"). OpenSpec has no delta form for
deleting a capability. The delta was amended to reduce `agent-lifecycle` to a
single retirement tombstone requirement ("Imperative agent lifecycle is retired"),
which is both valid and truthful and carries no `` `homonto agents` `` token (so
`spec-command-check` still passes). Re-validated: `openspec validate` clean,
`--all` 16/16. This is a delta-correctness fix discovered at the gate, not a
behavior change.

## Out of scope (recorded)

- `config.go:526` silently discarding `[agents]` after the fold — F35-adjacent,
  separate change.
- `docs/superpowers/*` historical residue mentioning `homonto agents` — F19,
  separate change.
