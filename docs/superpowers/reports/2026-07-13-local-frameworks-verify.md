# Verification — local-frameworks (E1 flagship: local framework resolution)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (catalog FS-aware + mergeFrameworkRoot; config local: + FrameworkCatalog + expansion; engine wiring) | PASS |
| 3 | Delta scenarios (local framework installs; non-builtin/non-local fails; MODIFIED F35 reconciled) | PASS |
| 4 | proposal goals (a local framework installs through the same validated path as a builtin) | PASS |
| 5 | E2E acceptance: `TestApply_LocalFrameworkSkillMaterialized` (local framework skill materialized by apply) | PASS |
| 6 | `go test ./... -race` (679) | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual — security-sensitive config validation) | PASS |

## Code review (standard, manual)
- **Config validation**: `validateFrameworkResources` accepts `builtin:<name>`/`local:<path>` and rejects every other source (F35 preserved for remote:/bare); skills/commands keep their `local:` plain-name rules (frameworks get their own path). Under the D1 trust model a `local:` framework is the user's own filesystem (as trusted as their homonto.toml), so no digest/traversal gate — consistent with `local:` skill sources.
- **Catalog FS-aware index**: base-only catalogs resolve every resource from the base FS → byte-identical (the whole catalog + engine + config suite, 679 tests, green). `mergeFrameworkRoot` reuses the manifest-schema/name/path/strict-conflict checks.
- **EnabledModelTools**: only builtin frameworks force model routes; since local: frameworks did not exist before, the builtin path is identical. Documented limitation (a local framework's model-routed commands don't force a [models] block) — a scoped follow-up, not a regression.
- **Spec**: framework-expansion F35 requirement MODIFIED to "non-builtin AND non-local fails"; the renamed reject-test is an intentional contract update E1 overturns.
- Delegated implementation independently re-verified: build + E2E + full -race + vet; gate test untouched; the FrameworkCatalog casing diagnostic was stale (build confirms exported + wired).

## Behavior / risk
Medium-scope cross-subsystem feature; builtin-only configs byte-identical (679 tests). Delivers E1's flagship local-framework capability end-to-end (D1). Remaining E1: remote/digest frameworks, [compat].homonto, capabilities — later/decision-gated phases.
