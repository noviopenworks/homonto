# Verification — exit-code-taxonomy (F50)

Light verification (tweak). All 6 checks PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Diff matches tasks (cli exit-code sink + plan/status flags + main.go) | PASS |
| 3 | `go build ./...` | PASS |
| 4 | `go test ./internal/cli/... -race` (33 tests) | PASS |
| 5 | No security issues (no Old/New leaked; opt-in flag, default unchanged) | PASS |
| 6 | Code review: `review_mode: off` — skipped per config | PASS (skipped) |

Also: `go vet ./...` clean, `openspec validate exit-code-taxonomy` valid, `cmd/onto` still builds (unchanged).

**Behavior:** `plan --exit-code` → 2 when changes/repins pending, else 0. `status --exit-code` → 3 drift / 2 pending / 0 clean. Without the flag, exit code is unchanged (0 on success, 1 on error). Only homonto's `main.go` changed; onto's is untouched.
