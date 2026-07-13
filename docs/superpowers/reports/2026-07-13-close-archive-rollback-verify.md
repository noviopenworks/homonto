# Verification — close-archive-rollback (X2/F4 onto close consistency)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (error-path rollback of archived flag; success path unchanged) | PASS |
| 3 | Delta scenarios (failed move leaves archived=false + change unmoved; success unchanged) | PASS |
| 4 | proposal goals (failed close no longer leaves marked-archived-but-not-moved) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | onto close suite (success + all refusal paths) | PASS |
| 7 | vet, build (incl. cmd/onto), openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- rollback closure (`st.Archived=false; Save`) runs on both MkdirAll and Rename failure, restoring the in-place onto-state.yaml to archived=false. statePath stays valid on failure (change dir unmoved). Success path unchanged: rename carries the archived:true state file into the archive dir.
- Bounded to the deterministic error path; a process kill between the save and the rename still has a window (documented out-of-scope; full crash-safety needs location-derived archived state). This matches the spec MODIFY (failed move rolls back).
- New failure-injection test (archive parent made a file → MkdirAll fails) proves the rollback; RED on prior code.

## Behavior / risk
Low-risk error-path fix; no success-path change. Closes F4's close-archive-ordering consistency. Remaining X2 (stateless Apply, transaction journals F42) unaffected.
