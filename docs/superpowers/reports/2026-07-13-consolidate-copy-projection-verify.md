# Verification — consolidate-copy-projection (F40 copy-mode follow-on)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (copyproj wraps copyfile; Name/Plan/Apply; adapters keep desired+pruneRoots) | PASS |
| 3 | Delta scenarios (adapters reconcile via core; prune-root guard preserved) | PASS |
| 4 | proposal goals (last duplicated adapter surface consolidated) | PASS |
| 5 | `go test ./... -race` | PASS (655) |
| 6 | Conformance + per-adapter copy-mode tests | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- copyproj reproduces the adapters' logic exactly: conflict error keyed by `tool`; LocalEdit nil-Content→Prune / Content→Update (.bak backed up first); refused prunes NOT in `pruned` so ownership retained and out-of-root file never deleted (F7); state Set/Delete on `subagentcopy.`+Name(dst).
- Adapters keep only copySubagentDesired + copyPruneRoots; thin planCopyOps/applyCopySubagents wrappers delegate. Plan emit uses copyproj.Name.
- internal/copyfile untouched; no `_test.go` modified.

## Behavior / risk
Pure refactor, no behavior change. Completes F40's adapter consolidation across all three surfaces (structured-doc, file-projection, copy-mode): claude 1037→714, opencode 999→731, behind shared structproj+jsoncodec+fileproj+copyproj cores.
