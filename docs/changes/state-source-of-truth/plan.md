# Plan: state-source-of-truth

Design: `design.md` (Status: Confirmed 2026-07-05). One commit per task.
TDD where testable logic exists (adapters/engine); direct for CLI wiring
proven by the final validation task.

## Task 1 — Add the `adopt` action + `plan.HasAdoptions`

- [x] done
- Files: `internal/adapter/adapter.go`, `internal/plan/plan.go`,
  `internal/plan/plan_test.go`
- Do: extend the `Change.Action` enum comment (adapter.go:18-23) to include
  `"adopt"`. Add `plan.HasAdoptions(sets) bool` (true iff any change action is
  `adopt`). `plan.Render` already switches only on create/update/delete, so
  `adopt` renders nothing — add a test asserting that. Leave `HasChanges`
  meaning "visible change" (unchanged).
- Verify: `rtk go test ./internal/plan/...` — new tests for `HasAdoptions`
  (true/false) and "adopt renders no line" pass.

## Task 2 — claude adapter: emit and apply `adopt`  (risk: high)

- [ ] done
- Files: `internal/adapter/claude/claude.go`,
  `internal/adapter/claude/*_test.go`
- Do: in `Plan` (claude.go:88-90), split the non-secret match branch —
  `inState → noop`, `!inState → adopt` (carry `New: want`). In `Apply`
  (before claude.go:180 noop skip), handle `adopt`: resolve `c.New`, then
  `st.Set("claude", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))`
  and `continue` — no file write. Per design "adapter Apply" + ADR
  `adopt-preexisting-resources-into-state`.
- Verify (TDD): failing test first — a declared MCP present on disk == desired,
  absent from state, Plan yields `adopt`; Apply records state, `.claude.json`
  byte-unchanged; then removal from config yields a `delete`. `rtk go test
  ./internal/adapter/claude/...`.

## Task 3 — opencode adapter: emit and apply `adopt`  (risk: high)

- [ ] done
- Files: `internal/adapter/opencode/opencode.go`,
  `internal/adapter/opencode/*_test.go`
- Do: in `planKey` (opencode.go:136-137) split non-secret match branch as in
  Task 2; in the plugin branch (opencode.go:74-75) emit `adopt` when
  `arrayHas` is true and the key is absent from state. In `Apply` handle
  `adopt` state-only (mirror Task 2; plugin `New = mustJSON(name)`). Keep
  claude/opencode in lockstep.
- Verify (TDD): failing tests first — pre-existing opencode setting AND plugin
  matching desired, absent from state → `adopt`; Apply records both, file
  unchanged apart from JSONC standardization; pruneable after removal.
  `rtk go test ./internal/adapter/opencode/...`.

## Task 4 — engine.Apply skips `adopt` in the resolve loop

- [ ] done
- Files: `internal/engine/engine.go`, `internal/engine/*_test.go`
- Do: extend the resolve-loop skip (engine.go:85) to
  `noop || delete || adopt` (adopt is non-secret; nothing to resolve). Confirm
  per-adapter state save still runs so adopted records persist.
- Verify: `rtk go test ./internal/engine/...` (adoption-through-engine test:
  Apply on an all-matching-but-unrecorded config persists adopted state).

## Task 5 — apply.go three-way reconcile flow

- [ ] done
- Files: `internal/cli/apply.go`, `internal/cli/*_test.go`
- Do: replace the single `!HasChanges` short-circuit (apply.go:42-45) with:
  no work → "No changes. Everything up to date."; only adoptions
  (`!HasChanges && HasAdoptions`) → `e.Apply(sets)` with no prompt, then
  "Reconciled N pre-existing resource(s) into state." (N = count of adopt
  changes); else render + prompt + apply. Per design "apply.go flow".
- Verify: `rtk go test ./internal/cli/...` — adoption-only run applies without
  prompt and prints the reconcile summary; mixed run still prompts.

## Task 6 — ObserveHashes on the interface + both adapters

- [ ] done
- Files: `internal/adapter/adapter.go`, `internal/adapter/claude/claude.go`,
  `internal/adapter/opencode/opencode.go`, adapter `*_test.go`
- Do: add `ObserveHashes(st *state.State) (map[string]string, error)` to the
  `Adapter` interface. claude: read via `current()`, hash each recorded key's
  on-disk value (`secret.Hash(jsonutil.Canonical(disk))`), omit absent keys.
  opencode: read the file once, extract each recorded `mcp.`/`setting.` value
  and hash; `plugin.<name>` present in the array → `hash(canonical(mustJSON
  (name)))`; omit absent. Only hashes escape (secret-safe). Per ADR
  `drift-from-disk-vs-state`.
- Verify (TDD): failing tests — recorded key matching disk → hash == `Applied`;
  edited on disk → differs; absent → omitted. `rtk go test ./internal/adapter/...`.

## Task 7 — engine.Status (drift = disk-vs-Applied, pending separate)  (risk: high)

- [ ] done
- Files: `internal/engine/status.go`, `internal/engine/status_test.go`
- Do: rewrite drift as `Status() (drift []string, pending int, err error)`.
  For each tool: `observed = adapter.ObserveHashes(state)`; per recorded key
  → absent = "<tool> <key> missing (deleted out of band)", hash≠`Applied` =
  "<tool> <key> drifted"; collect drifted keys. `pending` = count of `Plan()`
  visible changes (create/update/delete) whose (tool,key) ∉ drifted. Preserve
  adapter-skip warnings. Keep/replace the `Drift` name per design note.
- Verify (TDD): the gap test — recorded key, `homonto.toml` desired edited,
  disk unchanged → drift empty, pending == 1. Plus true-positive drift and
  missing. `rtk go test ./internal/engine/...`.

## Task 8 — status.go CLI prints drift + pending

- [ ] done
- Files: `internal/cli/status.go`, `internal/cli/*_test.go`
- Do: call `e.Status()`; print warnings, then drift lines; when `pending > 0`
  print "N config change(s) awaiting apply (run `homonto apply`)"; when both
  empty print "No drift."
- Verify: `rtk go test ./internal/cli/...`.

## Task 9 — Validation (the change proves itself)

- [ ] done
- Files: none (evidence only) — plus any test gaps found
- Do: full suite + manual smoke. Manual: scratch config with a pre-existing
  matching resource → `apply` adopts silently ("Reconciled …"); edit config →
  `status` shows pending not drift; edit disk out of band → `status` shows
  drift.
- Verify: `rtk go test ./...`, `rtk go vet ./...`,
  `rtk go build -o /tmp/homonto-build .`, `rtk go test -race ./...` all pass;
  capture the manual smoke output. Confirm claude/opencode parity and that
  secretsafety tests still pass.
