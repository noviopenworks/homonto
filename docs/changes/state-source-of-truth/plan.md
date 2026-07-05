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

- [x] done
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

- [x] done
- Files: `internal/adapter/opencode/opencode.go`,
  `internal/adapter/opencode/*_test.go`
- Do: in `planKey` (opencode.go:136-137) split non-secret match branch as in
  Task 2; in the plugin branch (opencode.go:74-75) emit `adopt` when
  `arrayHas` is true and the key is absent from state. In `Apply` handle
  `adopt` state-only (mirror Task 2; plugin `New = mustJSON(name)`). Keep
  claude/opencode in lockstep.
- Verify (TDD): failing tests first — pre-existing opencode setting AND plugin
  matching desired, absent from state → `adopt`; Apply records both; pruneable
  after removal. `rtk go test ./internal/adapter/opencode/...`. (Byte-unchanged
  file guarantee is proven in Task 3b, which adds the conditional write.)

## Task 3b — Apply writes a tool file only when a key in it changed (both adapters)  (risk: high)

- [x] done
- Files: `internal/adapter/claude/claude.go`, `internal/adapter/opencode/opencode.go`,
  adapter `*_test.go`
- Do: per design "Conditional tool-file writes". Track per-file "changed"
  (claude: `mjChanged` for `mcp.*`, `sjChanged` for `setting.*`/`plugin.*`;
  opencode: `docChanged` for `mcp.*`/`setting.*`/`plugin.*`) set on
  create/update/delete; `adopt`/`noop` never set it. Call `WriteAtomic` for a
  file only when its flag is set. `skill.*` symlink work is unaffected. This
  makes adoption literally write no tool file, and stops adopt/noop-only
  applies from reformatting/comment-stripping. Retrofits the claude adopt from
  Task 2.
- Verify (TDD): failing tests first — an adopt-only apply against
  `opencode.jsonc` containing a COMMENT leaves the file byte-identical
  (comment preserved); against a non-standard-formatted `.claude.json` leaves
  it byte-identical. Full suite green (root-cause any test that relied on
  incidental file creation/standardization on a no-key-change apply).
  `rtk go test ./...`.

## Task 4 — engine.Apply skips `adopt` in the resolve loop

- [x] done
- Files: `internal/engine/engine.go`, `internal/engine/*_test.go`
- Do: extend the resolve-loop skip (engine.go:85) to
  `noop || delete || adopt` (adopt is non-secret; nothing to resolve). Confirm
  per-adapter state save still runs so adopted records persist.
- Verify: `rtk go test ./internal/engine/...` (adoption-through-engine test:
  Apply on an all-matching-but-unrecorded config persists adopted state).

## Task 5 — CLI wiring: HasChanges = visible-only + apply.go three-way flow

- [x] done
- Files: `internal/plan/plan.go`, `internal/plan/plan_test.go`,
  `internal/cli/apply.go`, `internal/cli/*_test.go`
- Do:
  1. Fix `plan.HasChanges` to mean **visible change only** — true iff any
     action is create/update/delete (exclude `adopt` as well as `noop`). Its
     "!= noop" form silently started counting `adopt` when the action was
     added; restore the contract. With this, `internal/cli/plan.go` needs no
     change (adopt-only → `!HasChanges` → "No changes", matching the gate:
     plan stays silent about adoption). Update/confirm plan_test.
  2. Replace the single `!HasChanges` short-circuit in `internal/cli/apply.go`
     (apply.go:42-45) with a three-way branch: `!HasChanges && !HasAdoptions`
     → "No changes. Everything up to date."; `!HasChanges && HasAdoptions` →
     `e.Apply(sets)` with NO prompt, then "Reconciled N pre-existing
     resource(s) into state." (N = count of adopt changes across sets); else
     render + prompt + apply (adoptions ride along). Per design "apply.go flow".
- Verify: `rtk go test ./internal/plan/... ./internal/cli/...` — adoption-only
  `apply` applies without prompt and prints the reconcile summary; adoption-only
  `plan` prints "No changes"; a mixed run still renders + prompts.

## Task 6 — ObserveHashes on the interface + both adapters

- [x] done
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

## Task 7 — engine.Status (drift = disk-vs-Applied, pending separate) + status.go CLI  (risk: high)

- [ ] done  <!-- absorbs former Task 8 (CLI): removing Drift() would break cli/status.go, so wire both in one green commit -->
- Files: `internal/engine/status.go`, `internal/engine/status_test.go`,
  `internal/cli/status.go`, `internal/cli/*_test.go`
- Do: replace `engine.Drift()` with `Status() (drift []string, pending int,
  err error)`. Call `e.Plan()` (sets + warnings). For each adapter: `observed
  = adapter.ObserveHashes(e.State)` (on error → warn, skip that tool). Per
  recorded key `st.Keys(tool)`: absent from `observed` → "<tool> <key> missing
  (deleted out of band)"; `observed[key] != Entry.Applied` → "<tool> <key>
  drifted"; add BOTH to the drifted set. `pending` = count of `sets` visible
  changes (create/update/delete) whose (tool,key) ∉ drifted set. Update the
  existing engine `Drift` tests to `Status`. Then rewrite `cli/status.go`:
  print warnings, drift lines; if `pending > 0` print "N config change(s)
  awaiting apply (run `homonto apply`)"; if drift and pending both empty print
  "No drift." Update cli status tests.
- Verify (TDD): the GAP test — recorded key, `homonto.toml` desired edited,
  disk unchanged → drift empty, pending == 1. Plus true-positive drift, missing,
  and no-drift-no-pending. `rtk go test ./internal/engine/... ./internal/cli/...`.

## Task 8 — (merged into Task 7)

- [x] 8 merged into Task 7 (CLI status wiring committed together to keep the
  build green when `Drift` is replaced).

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
