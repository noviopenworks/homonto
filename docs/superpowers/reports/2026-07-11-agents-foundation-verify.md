# Verification Report: agents-foundation (v2 #1)

- **Change**: `agents-foundation` — `[agents.<name>]` lifecycle model + read-only `homonto agents list`
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (2 capabilities: config-model + new agent-lifecycle)
- **Result**: PASS — final review found no bugs (one forward-looking hardening applied)

## Scope

`internal/config/config.go` (`Agent` type + `TargetsOrAll`/`ModeOrDefault` + `Config.Agents` + `validateAgents`), `internal/cli/agents.go` (`agentsCmd` + read-only `list`), root registration, tests, README + roadmap. Read-only foundation; no projection/lockfile/mutation.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 model, D2 validation, D3 list command) | PASS |
| 3 | Matches Design Doc (exact Agent type, validateAgents, agents.go) | PASS |
| 4 | All delta-spec scenarios pass (config-model + agent-lifecycle) | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| parse agent declaration | `TestAgentsParseFullDeclaration` | PASS |
| defaults (unpinned/both/link) | `TestAgentDefaults` | PASS |
| invalid source rejected | `TestAgentsRejectInvalidSource` | PASS |
| invalid mode rejected | `TestAgentsRejectInvalidMode` | PASS |
| unknown target rejected | `TestAgentsRejectUnknownTarget` | PASS |
| traversal name rejected (hardening) | `TestAgentsRejectTraversalName` | PASS |
| list declared agents (sorted) | `TestAgentsListReportsDeclaredAgents` | PASS |
| no agents declared | `TestAgentsListNoAgents` | PASS |
| list is read-only | `TestAgentsListIsReadOnly` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 364 passed, 23 packages |
| `go test -race ./internal/config/... ./internal/cli/...` | 63 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary)

A config with `[agents.review]` (pinned builtin, both targets, mode=copy) +
`[agents.explore]` (local, unpinned) → `homonto agents list` printed both SORTED
(explore, review) with source/version/targets/mode, `version=unpinned` and
`mode=link` defaults correct. Empty config → `No agents declared.`. Invalid
source `https://x` → clear load error, exit 1. `homonto agents` (no subcommand) →
help. No files created (read-only).

## Code review (review_mode: standard) — no bugs

The final review verified all seven axes correct: validation IS wired into
`config.Load` (the only load path) and its error returned; source/mode/target/
name all validated; `list` is genuinely read-only (config.Load + print only, no
engine, no writes); defaults render correctly; output is `sort.Strings`-sorted
(deterministic); `--config` inherited correctly; nil `[agents]` map ranged safely;
tests assert sorted order + defaults + zero files created. Its one forward-looking
note — agent names should use the stricter `validateResourceName` (rejecting
`../`, `/`) before projection lands — was **applied proactively** (agents are
projected to files named by the agent name in later v2 increments) with a
traversal-name test.

## Conclusion

Verification PASS. First (foundation) increment of roadmap v2 Agent Lifecycle:
the `[agents.<name>]` declaration model + read-only inspection. Deferred to later
increments: add/update/pin/doctor/migrate, the lockfile + installed state,
compatibility checks, three-way-merge, remote sources, and `[agents]`-vs-
`[subagents]` reconciliation.
