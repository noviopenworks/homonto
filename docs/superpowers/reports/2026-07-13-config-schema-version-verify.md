# Verification — config-schema-version (X3/F37 config forward-safety)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (SchemaVersion field + const + Load fail-closed check, mirrors state) | PASS |
| 3 | Delta scenarios (future version rejected; absent/current load unchanged) | PASS |
| 4 | proposal goals (config plane now has state-plane forward-safety) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | config pkg + full suite | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- Guard runs immediately after toml.Unmarshal, before the agents fold and any adapter/plan/apply — fail-closed with an "upgrade homonto" message. Absent/0 treated as current (backward-compatible; every existing config + all test fixtures load unchanged, verified by the full suite). Mirrors state.Load's check exactly.
- `schema_version,omitempty` so a legacy config never gains the key; homonto never rewrites homonto.toml, so nothing stamps it (authored by users/future tooling).
- No behavior change for any config without a future version.

## Behavior / risk
Low-risk additive field + one load guard. Closes F37's config half (state half already shipped). Remaining X3 (ToolID capability registry F33/F34, config-loading phase split F43, non-waivable finding classes F11/F12) is larger and design-first.
