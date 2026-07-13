# Verification — crash-safe-catalog-materialize (X2/F47 skill-dir staging)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (stage-then-swap per skill; commands/subagents unchanged) | PASS |
| 3 | Delta scenarios (failure never corrupts dst; success writes identical content) | PASS |
| 4 | proposal goals (partial skill dir can no longer pass allSkillDirsExist) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | catalog + engine materialize suites (byte-identity regression) | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- Staging is a sibling `<dst>.staging` under the same control-plane root, so `WriteControlPlane`'s no-follow guarantee is preserved. Step 1 `RemoveAll(staging)` discards any prior-crash leftover; walk writes into staging; on success `RemoveAll(dst)`+`Rename(staging,dst)`; on any walk error dst is untouched and partial staging is dropped.
- Crash windows: mid-walk → dst is the prior complete version; between RemoveAll(dst) and Rename → dst absent (not partial) so allSkillDirsExist re-materializes next run.
- Success-path bytes identical (all existing materialize tests green); commands/subagents untouched (already atomic).
- TDD test drives the leftover-staging cleanup (fails on old code) + asserts no stale content leaks.

## Behavior / risk
Low-risk, localized to one function; no success-path behavior change. Closes the F47 catalog-materialization destructiveness. Remaining X2 (stateless Apply, transaction journals, close/archive validation) unaffected.
