## Why

ROADMAP E2 / finding F49: `homonto plan/apply/status/doctor/import` set no cobra
`Args` constraint, so a stray positional is silently ignored — a user running
`homonto apply production.toml` (expecting that file to be used) gets the default
config with no error. Config is selected only via `--config`.

## What Changes

- `plan`, `apply`, `status`, `doctor`, `import` set `Args: cobra.NoArgs` so a
  stray positional errors instead of being silently dropped. `init` keeps
  `MaximumNArgs(1)` (its optional target dir).

## Impact

- **Code:** `internal/cli/{plan,apply,status,doctor,import}.go` + a test.
- **Spec:** `cli-commands` delta (commands reject unexpected positional args).
- **Out of scope:** the rest of E2 (JSON output, exit-code taxonomy) and X1-X3.
