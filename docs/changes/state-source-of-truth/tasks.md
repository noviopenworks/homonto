# Tasks: state-source-of-truth

<!-- Refined from design.md (Approach 1). onto-build checks off, one commit per item. -->

## 1. `adopt` action foundation

- [x] 1.1 Add `adopt` to the `Change.Action` enum comment
      (`internal/adapter/adapter.go`) and `plan.HasAdoptions(sets)` in
      `internal/plan/plan.go` (Render already omits unknown actions — assert
      `adopt` renders no line).

## 2. Adoption in the adapters

- [x] 2.1 claude: Plan emits `adopt` for a declared non-secret key present on
      disk == desired but absent from state (split the non-secret match branch
      `inState → noop` / `!inState → adopt`); Apply records state for `adopt`
      without writing the tool file.
- [x] 2.2 opencode: same in `planKey` and the plugin branch; Apply `adopt`
      state-only write. Keep claude/opencode in lockstep.
- [x] 2.3 `engine.Apply` resolve loop skips `adopt` (non-secret; nothing to
      resolve).
- [x] 2.4 Both adapters: `Apply` writes a tool file only when a managed key in
      it changed (create/update/delete); `adopt`/`noop`-only apply leaves the
      file byte-unchanged (comments preserved). Retrofits claude from 2.1.

## 3. apply.go reconcile flow

- [x] 3.1 Three-way branch in `internal/cli/apply.go`: no work → "No changes";
      only adoptions → state-only apply, no prompt, "Reconciled N pre-existing
      resource(s) into state."; else render + prompt + apply (adoptions ride
      along).

## 4. Drift decoupled from Plan

- [x] 4.1 Add `ObserveHashes(st) (map[string]string, error)` to the `Adapter`
      interface and both adapters (secret-safe: only hashes escape; opencode
      plugins map array-presence to `hash(canonical(mustJSON(name)))`).
- [x] 4.2 Rewrite `engine` drift as `Status() (drift []string, pending int,
      err error)`: drift = disk hash ≠ `Applied` (or missing); pending = Plan
      visible changes whose key is not drifted. Preserve warning-on-skip.
- [x] 4.3 `internal/cli/status.go`: call `Status`, print drift lines + "N
      config change(s) awaiting apply" when pending > 0.

## 5. Specs (delta — merged at close)

- [ ] 5.1 `specs/apply-pipeline.md` delta already drafted (ADDED "State
      adoption on apply", MODIFIED "Drift detection") — keep in sync if build
      diverges.
- [ ] 5.2 `specs/tool-adapters.md` delta already drafted (ADDED "Adapters
      adopt pre-existing matching keys") — keep in sync if build diverges.

## 6. Validation

- [ ] 6.1 Tests: adoption records state + tool file unchanged + pruneable;
      adoption-only apply (no prompt, reconcile summary, second apply no-op);
      drift true positive (out-of-band disk edit); drift true negative (config
      edit, disk unchanged, NOT drift, pending=1); missing key; claude/opencode
      parity incl. opencode plugin; secret behavior unchanged.
- [ ] 6.2 `go test ./...`, `go vet ./...`, `go build`, `go test -race ./...`
      pass; manual `status`/`apply` smoke on a scratch config demonstrating the
      pending-vs-drift distinction and silent adoption.
