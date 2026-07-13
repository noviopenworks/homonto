## Why

ROADMAP E2 / finding F45: `plan` prints "No changes. Everything up to date." and
`status` prints "No drift." — and exit 0 — even when adapter warnings were emitted
(a skipped/degraded adapter means coverage was incomplete). `apply` already guards
this with its `skipped()` error; `plan`/`status` do not. A clean conclusion after
incomplete coverage misleads automation.

## What Changes

- A shared `coverageComplete(warnings)` helper returns a non-zero error when any
  warning was emitted. `plan` and `status` call it instead of printing a clean
  "up to date" / "No drift" conclusion when warnings exist; `apply`'s existing
  `skipped()` is refactored onto the same helper.

## Impact

- **Code:** `internal/cli/{plan,status,apply}.go` + a helper + test.
- **Spec:** `cli-commands` delta (no clean conclusion after incomplete coverage).
- **Out of scope:** the exit-code taxonomy / `--output json` (rest of E2).
