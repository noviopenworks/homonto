# Verification — framework-compat-homonto (E1 [compat].homonto)

Full verification. PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (catalog version-agnostic Compat; engine checks vs HomontoVersion; SatisfiesLoose strips pre-release) | PASS |
| 3 | Delta scenarios (incompatible fails fail-closed; compatible/unconstrained load) | PASS |
| 4 | proposal goals (shared frameworks fail loud on version mismatch) | PASS |
| 5 | E2E: incompatible framework fails Plan; compatible dev-build loads | PASS |
| 6 | `go test ./... -race` | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- Catalog stays version-agnostic (stores Framework.Compat; no cli import). Engine.checkFrameworkCompat runs at the START of Plan (choke point for cli/status/doctor), fails closed before projection; empty HomontoVersion skips (tests unaffected); frameworks with no [compat] unconstrained. Maps each declared framework to its catalog name via the exported config.FrameworkCatalogName.
- SatisfiesLoose strips a `-prerelease`/`+build` suffix before the comparator so 0.1.0-dev satisfies >=0.1.0 (a dev build of a version counts as that version) — table-tested.
- Used Engine.HomontoVersion field (cli sets it after Build at 5 sites) instead of a Build signature change, avoiding a 28-test-site ripple; existing tests (no HomontoVersion) unchanged. Exported SatisfiesLoose + FrameworkCatalogName for the engine (no import cycle).

## Behavior / risk
Low; additive fail-loud check gated on a version the CLI stamps. Completes E1's compatibility mechanisms. Remaining E1: F38 (doc honesty of [plugins]).
