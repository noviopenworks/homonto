## Why
ROADMAP E2 / finding F50 (completes E2): every error exits 1 and every success
exits 0, so automation cannot distinguish "no changes" from "changes pending" from
"drift" by exit code. Add a taxonomy, but OPT-IN via `--exit-code` so default
behavior (and all existing automation) is unchanged.
## What Changes
- `homonto plan --exit-code` and `status --exit-code`: with the flag, the command
  exits with a documented code — 0 clean, 2 pending changes, 3 drift — instead of
  always 0. Without the flag, behavior is unchanged (0 on success, 1 on error).
- A `cli.Execute(args) int` entrypoint carries the code; `main.go` uses it. Errors
  still exit 1.
## Impact
- **Code:** `internal/cli` (Execute wrapper + exit-code sink + plan/status flags), `main.go` + tests.
- **Spec:** `cli-commands` delta (opt-in exit-code taxonomy).
- **Out of scope:** codes for doctor-findings / apply beyond this minimal set; onto's binary.
