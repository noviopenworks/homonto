# Verification Report — cli-reject-stray-args

**Date:** 2026-07-13 · ROADMAP E2 / F49 · Comet tweak · Result: PASS

- `plan`/`apply`/`status`/`doctor`/`import` set `cobra.NoArgs` — a stray positional
  (e.g. `homonto apply production.toml`) now errors naming the arg instead of being
  silently ignored; `init` keeps its optional dir. `d4e2f2f`.
- Test `TestPositionalFreeCommands_RejectStrayArg` asserts each rejects the stray arg by name.

## Evidence
`go test ./internal/cli/... -race` OK; vet clean; build OK; `openspec validate --all` 16/16.
First slice of E2 (F49); the rest of E2 (JSON output, exit-code taxonomy) and X1-X3/E1/E3/E4 remain.
