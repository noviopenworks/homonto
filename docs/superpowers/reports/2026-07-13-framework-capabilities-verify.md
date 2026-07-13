# Verification — framework-capabilities (E1 D2 capability model)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (name@major parse; validateCapabilities post-merge; provider set) | PASS |
| 3 | Delta scenarios (unresolved fails; satisfied/absent load; cross-source resolves) | PASS |
| 4 | proposal goals (interface-based dependency; real consumer) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | catalog suite + real embedded catalog (comet requires / openspec provides) | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- `parseCapability` (name@major, non-empty name + non-negative int major) — malformed fails loud both as provided and required. `validateCapabilities` runs after ALL sources merge (base + overlays), so a cross-source capability (overlay provider) resolves; unresolved requirement errors naming the framework + capability. Wired into both `LoadOverlays` and `LoadWithLocal`.
- Multiple providers allowed (interface, not resource); frameworks without capabilities unchanged (catalog suite green). Real consumer: openspec provides spec-workflow@1, comet requires it — the embedded catalog loads, a test proves an unresolved requirement fails.
- Additive; no behavior change for capability-free configs.

## Behavior / risk
Low-risk additive parse + a load-time resolution pass. Meaningful now that frameworks can be shared (local/remote) — depend on an interface, not a name. Remaining E1: `[compat].homonto` (version injection), F38 (D4).
