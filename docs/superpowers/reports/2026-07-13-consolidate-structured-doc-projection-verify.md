# Verification — consolidate-structured-doc-projection (F40 structured-doc slice)

Full verification (full workflow + delta spec). PASS.

## Checks
| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Implementation matches design.md/Design Doc (structproj per-doc namespaces, shared JSON codec, file-projection out of scope) | PASS |
| 3 | Delta spec scenarios satisfied (claude routes settings/.claude.json keys through core; opencode routes opencode.json keys; shared JSON codec used by both) | PASS |
| 4 | proposal.md goals met (duplicated structured-doc projection removed; behavior preserved) | PASS |
| 5 | Full suite `go test ./... -race` | PASS (645) |
| 6 | Adapter suites `-race` | PASS (137) |
| 7 | Conformance suite (drift, malformed, secret non-leak, foreign-content) all 3 adapters | PASS |
| 8 | `go vet`, `go build`, `openspec validate --all` (16/16) | PASS |
| 9 | Code review (standard): correctness/security/edge-cases | PASS (see below) |

## Code review (standard, manual — security-sensitive diff)
Reviewed the delegated migration in full:
- **Delete de-duplication**: generic prune loops narrowed (`filePrefix` in claude; `managedPrefix` = plugin.+file in opencode) so structproj's own prefix deletes are not double-emitted. `plugin.` (opencode, array-based) correctly stays in the generic loop for orphan pruning; structproj does not own it.
- **Secret safety**: `Old: adapter.SecretRedaction` paths preserved; conformance secret-non-leak test green; codec never reads on-disk value for secret-bearing desired (structproj already enforced this identically).
- **Byte-identity**: settings.json prefixes applied in lexicographic order (marketplace<plugin<pluginconfig<setting) reproducing the prior single-sorted-loop insertion order; per-doc write-gating (mjChanged/sjChanged) preserved; `EnsureRoot` idempotent on a valid doc when threading.
- **Scope**: file-projection (symlinks, copy-subagents) and opencode plugin-array logic untouched; no `_test.go` modified.

## Behavior / risk
Pure refactor, no behavior/schema change. Duplicated projection control-flow removed from both adapters (claude 1037→948, opencode 999→954) in favor of the shared structproj core + one 66-line jsoncodec shared by both JSON adapters. File-projection consolidation is a documented follow-on.
