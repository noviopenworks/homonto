# Verification Report: agents-builtin-source (v2 #6a)

- **Change**: `agents-builtin-source` — resolve `builtin:<name>` agent sources from the embedded catalog
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (catalog + cli; agent-lifecycle add/update MODIFIED)
- **Result**: PASS — final review found no CRITICAL/IMPORTANT bugs

## Scope

`internal/catalog/catalog.go` (`SubagentContent`), `internal/cli/agents.go`
(`resolveAgentSource` + `agentMode` helpers; wired into add/update/doctor/list).
`local:` behavior unchanged; remote still rejected.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 SubagentContent, D2 resolver, D3 wiring, D4 builtin+link) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass (add + update MODIFIED) | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| catalog SubagentContent known/unknown | `TestSubagentContentReadsBuiltin` | PASS |
| add a builtin agent | `TestAgentsAddBuiltinInstallsCatalogContent` | PASS |
| builtin defaults to copy (no mode) | `TestAgentsAddBuiltinDefaultsToCopy` (E2E-caught) | PASS |
| unknown builtin is an error | `TestAgentsAddUnknownBuiltinIsError` | PASS |
| builtin + link is an error | `TestAgentsAddBuiltinLinkIsError`, `...Update...` | PASS |
| doctor healthy for a builtin | `TestAgentsDoctorBuiltinHealthy` | PASS |
| builtin update idempotent | `TestAgentsUpdateBuiltinIsIdempotent` | PASS |
| local: add/update/doctor unchanged | full prior agent suite | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 436 passed, 26 packages |
| `go test -race ./internal/catalog/... ./internal/cli/...` | 77 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary)

`[agents.cr] source="builtin:code-reviewer"` (no mode) → `agents add cr` installed
the catalog's code-reviewer content into both tool agent dirs (verified byte-equal
to `catalog/subagents/code-reviewer.md`), recording `mode=copy`; `agents doctor`
→ `healthy`; `agents list` → `mode=copy` (effective, not the raw link default).
`source="builtin:not-a-real-agent"` → "unknown builtin agent" error; an explicit
`builtin: mode="link"` → the builtin-copy-only error. `local:` agents (copy and
link) still install/merge end-to-end.

## Code review (review_mode: standard) — no CRITICAL/IMPORTANT bugs; one MINOR applied

The review verified: `agentMode` correctness (builtin+link→error, builtin+unset→
copy, local→ModeOrDefault) and its consistent use in the copy/link switch AND the
lockfile `Mode` record in both add and update; `resolveAgentSource` never returns
`(nil,nil)` (no empty-file install), `catalog.New()` error handled, unknown
builtin → clear error; the link branch (deriving `srcPath` from a `local:` source)
runs only for local agents (builtin coerced to copy, so its unused `srcPath` is
never read); doctor drift via the resolver with unresolved→finding (no crash);
`SubagentContent` return contract; builtin-name traversal safe (pure map lookup, no
filesystem path built from a builtin name); no local: coverage dropped. The one
MINOR — `agents list` showing the raw `link` default for a no-mode builtin — was
**applied** (list now shows the effective mode). An E2E-caught issue (no-mode
builtin hitting the link error) was fixed during build via `agentMode` + a
regression test before the review.

## Conclusion

Verification PASS. v2 #6a: `builtin:` agent source resolution from the embedded
catalog — the full agent lifecycle (add/list/doctor/update[--all], three-way
merge) now works for bundled agents as well as local ones. Remaining v2:
**remote** sources (explicit first-release non-goal), plus polish (compatibility
checks, orphan/target pruning, per-agent scope, blob GC, `--markers`,
`[agents]`-vs-`[subagents]` reconciliation).
