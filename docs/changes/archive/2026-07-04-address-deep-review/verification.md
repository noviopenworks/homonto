# Verification Report: address-deep-review

- **Date:** 2026-07-04
- **Mode:** full (workflow full; product + framework changes; ~40 files in range)
- **Range:** bf2203e..HEAD on `feature/20260704/address-deep-review`
- **Result: pass** (2 recorded deviations)

## Scenario evidence

Every scenario across the five delta specs was checked; the load-bearing
ones were reproduced empirically against the freshly-built binary in a fake
`$HOME` (never the real one). Two adversarial rounds ran; round-2 items 1–8
(all empirical fixes) were re-verified before the skeptic hit a session
limit, and the remaining mechanical checks were run by the coordinator.

| Requirement / Scenario | Verdict | Evidence |
|---|---|---|
| tool-adapters: Claude MCP schema fidelity | pass | e2e: apply → `~/.claude.json` `{"type":"stdio","command":"npx","args":[...]}`; command is a JSON string; conformance fixture `testdata/real-claude.json` consumed by `TestApplyOntoRealClaudeJSONPreservesSchema` |
| tool-adapters: Declarative pruning | pass | e2e: removed MCP + skill → plan `- mcp.brave / - skill.demo`, apply removed both from disk + state, next plan "No changes"; drift-not-orphan: unmanaged disk keys survive de-declaration (`TestClaudeDriftIsNotMistakenForOrphan`) |
| tool-adapters: Injection-safe key handling | pass | e2e: `a.b*c` lands as one literal key, converges, and deletes cleanly; index-like/empty names now rejected at config load (`error: … would be treated as a JSON array index`) — target file byte-identical |
| tool-adapters: Deterministic plan output | pass | e2e: two consecutive plans byte-identical (`diff` empty); `sort.SliceStable` + 20-iter unit test |
| apply-pipeline: per-adapter state save | pass | `TestPartialApplyPersistsEarlierAdapterState` — claude's record survives opencode failure; error names the tool (`%s: %w` wrap) |
| apply-pipeline: single resolution | pass | counting `pass` stub: 1 call for a token shared across 2 keys + 2 phases; no cross-backend cache collision |
| apply-pipeline: unknown-provenance redaction | pass | state-absent drift → plan shows `«secret»` for both secret and plain keys; no planted value greppable |
| cli-commands: import string+args | pass | import preserves `command`+`args`; legacy array tolerated; malformed file → warning (no silent empty import); url/http server skipped with warning |
| cli-commands: version reporting | pass | `-ldflags -X …cli.Version=1.2.3` → `homonto version 1.2.3`; dev default otherwise |
| secret-references: modes | pass | new files 0600; existing 0600/0644 preserved; fsync before rename; symlinked target written through (`EvalSymlinks`) |
| onto-workflow: preflight warns-not-halts | pass | no HALT in any onto skill (grep); guide + sub-skills reworded; grounding fallback recorded in notes |
| onto-workflow: preset scope + thresholds | pass | tweak ≤5 entry / >5 upgrade, fix >5 non-test — consistent across 4 files; no `3+`/`5+` contradictions in live files |
| onto-workflow: close ADR-link rewrite | pass | onto-close step 2 rewrites `adr/<slug>.md` → `docs/adr/NNNN` before archive |

## Design conformance

All seven review priority items implemented except pushing to origin
(explicit non-goal). One design-time detail was corrected during build:
the escape set needed `@ | #` beyond `. * ? \` (empirically verified
against real sjson) — recorded in Task 4's return. Two Key decisions map
to the two ADRs (0007 errata + the new preflight-warns ADR draft).

## Adversarial pass

- Round 1 (conformance + robustness skeptics, empirical): FAIL — ~13
  findings (2 HIGH: index-name corruption, import parse silence; plus
  url-import loss, symlink-replaced-by-file, adapter-naming, tweak
  off-by-one, stale preflight-guarantee lines, drift-on-deletion, plugin
  scalar guard, relink dead-end). All fixed (commits e0ea17c + d24e656).
- Round 2 (focused skeptic, empirical): items 1–8 all COULD-NOT-REFUTE
  before a session limit stopped it; coordinator completed items 9–10
  (markdown coherence + regression) — clean.

## Regression

`go vet ./...` clean; `go test ./...` → **92 passed, 15 packages, 0
failed** (fresh). Self-containment grep over `content/skills/` → no
matches. Derivation tables byte-identical. Delta lint: all MODIFIED names
match living-spec headings exactly; every ADDED/MODIFIED requirement has
SHALL/MUST on its first line and ≥1 WHEN/THEN scenario.

## Deviations (accepted)

1. **Two-phase token equivalence for env/pass names containing quotes or
   backslashes** — phase-1 pre-resolve operates on raw JSON text, phase-2
   on decoded leaves; a token *body* with a JSON-escaped char could differ
   between phases. Exotic (requires a secret reference name containing `"`
   or `\`); not worth the complexity now. Recorded for a future change.
2. **Kill -9 between CreateTemp and rename strands a `.homonto-*` temp
   file** — pre-existing atomic-write property, not introduced here; the
   defer cleans only on error return. Cosmetic; documented.
