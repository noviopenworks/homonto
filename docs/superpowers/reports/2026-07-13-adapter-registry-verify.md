# Verification — adapter-registry (X3/F33 tool-id-keyed adapter registry)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (registry Deps/Factory/Registry/Builtins; engine wires via it) | PASS |
| 3 | Delta scenarios (engine builds every registered adapter in order; adding one is a registration) | PASS |
| 4 | proposal goals (engine decoupled from concrete adapter constructors) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | engine + conformance + adapter suites | PASS |
| 7 | vet, build (no import cycle), openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- `Builtins()` returns a fresh registry per call (no global mutable state / init ordering); registers claude/opencode/codex in order with the exact same options as the prior hardcoded literal (With* chain preserved for claude/opencode; codex just Home). `Build` constructs in registration order → identical adapter list. `Register` panics on a duplicate id (startup programming-error guard).
- Import direction: registry → {adapter, claude, opencode, codex}; adapters → adapter (not registry). No cycle (build confirms). engine no longer imports the concrete adapters.
- Behavior-identical: same three adapters, same order, same options — pinned by engine + conformance + adapter suites.

## Behavior / risk
Low-risk wiring refactor, no behavior change. Adding a built-in adapter is now one Register line in Builtins(). Remaining X3: F34 interface-type generalization (config/secret/state), config-loading phase split (F43), non-waivable finding classes (F11/F12) — larger design-first.
