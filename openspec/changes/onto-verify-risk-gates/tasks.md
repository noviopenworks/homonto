# Tasks — onto-verify-risk-gates

## 1. onto-verify skill copy
- [ ] F11: the scale check's `full` trigger includes a security-sensitive-surface
      diff regardless of size; `light` requires no such surface. F12: adversarial
      triage declares security/data-loss/failed-core-acceptance findings CRITICAL
      and non-waivable in any mode. No-slop prose; catalog still loads.

## 2. Verify
- [ ] `go test ./... -race`, build, `openspec validate --all` green (the catalog
      loads the edited skill); prose is tight (onto-no-slop).
