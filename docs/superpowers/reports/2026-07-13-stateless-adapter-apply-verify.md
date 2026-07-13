# Verification — stateless-adapter-apply (X2 Apply derives from config)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (Apply takes cfg; shared expand() in Plan+Apply; codex ignores; engine passes e.Cfg) | PASS |
| 3 | Delta scenario (Apply correct without prior-Plan instance state) | PASS |
| 4 | proposal goals (hidden 'Plan-first' precondition removed) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | conformance + all adapter/engine tests | PASS |
| 7 | vet, build, openspec validate --all | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- `expand(cfg)` reproduces Plan's original entry-expansion verbatim (skill/command/subagent → a.skills/commands/subagents); called as the first statement of both Plan and Apply in claude+opencode. Same cfg → identical entries → identical links/files/state. Codex ignores cfg (`_ *config.Config`), body unchanged (ChangeSet-driven). engine.Apply passes e.Cfg.
- Test migration (delegated): ~135 direct adapter Apply call sites updated to pass each test's matching config (verified per-ChangeSet: cs0→c, cs1→c1, csScratch→c2, etc.); the failingAdapter mock signature updated. Spot-checked diffs are Apply-call-only — NO assertion changes. Confirmed via git diff that no test logic changed.
- Independently re-ran build + vet + full -race after the delegated change (LSP diagnostics were stale pre-final-edit; go vet on the conformance package confirmed clean).

## Behavior / risk
Pure structural refactor, no behavior change. Removes Apply's hidden dependence on prior-Plan instance state (a named X2 concern). Remaining X2: transaction journals (F42), driving Apply purely from the ChangeSet.
