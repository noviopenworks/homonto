# Verification — onto-verify-risk-gates (X3 F11/F12)

Light verification (tweak — a focused skill-copy change with a delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Diff matches tasks (onto-verify SKILL.md F11 scale trigger + F12 triage; onto-binary delta) | PASS |
| 3 | Catalog loads the edited skill (`go test ./internal/catalog -race`) | PASS |
| 4 | Full suite `go test ./...` | PASS |
| 5 | build, `openspec validate --all` (16/16) | PASS |
| 6 | Prose is no-slop (tight, matches the skill's terse style; no filler) | PASS |
| 7 | Delta scenarios (small security change → full; critical class non-waivable) reflected in the copy | PASS |

## Notes
- F11: the `full` trigger now includes a security-sensitive-surface diff (secret resolution, remote fetch/verify, file deletion/pruning, permission/ownership) regardless of file count; `light` requires no such surface. Scale keys on risk, not just size.
- F12: the adversarial triage declares security/data-loss/failed-core-acceptance findings CRITICAL and non-waivable in any mode; only lower-severity findings are skip-eligible; a non-waivable-class finding still blocks even when a skeptic pass is skipped for lack of dispatch.
- Per B1, this judgment lives in the onto-verify skill (the agent's risk assessment), not the binary (which enforces the presence/shape of the verification result). No Go/binary change.
