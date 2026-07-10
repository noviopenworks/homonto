# Comet Design Handoff

- Change: onto-phase-gates
- Phase: design
- Mode: full
- Context hash: 82e84b43ecc0af60d4c127a6580459ffa293a07a7d166f46c3186c97e8d5312c

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-phase-gates/proposal.md

- Source: openspec/changes/onto-phase-gates/proposal.md
- Lines: 1-61
- SHA256: 2387f4aab359848b8a235eaddc6bb31a76b1423f8fb7e9ebeaef5d93cbfa9c41

```md
## Why

onto #1 (foundation), #2 (`onto init`), and #3a (`onto new` + skeleton
validation) are archived. The onto binary can create a change and validate its
skeleton, but it cannot yet ADVANCE a change through the workflow. Per the
dual-binary design the binary enforces structural invariants: "phase transitions
happen only through valid gates" and "dirty worktrees block close/archive". This
change (#3b, the second sub-increment of the onto workflow engine) adds gated
phase advancement.

## What Changes

- Extend `internal/ontostate.RequiredArtifacts(phase)` with per-phase supersets:
  `open` → onto-state.yaml, proposal.md, tasks.md; `design` → + design.md;
  `build` → + plan.md; `verify`/`close` → + verification.md. `ValidateSkeleton`
  (from #3a) automatically tightens as a change advances.
- Add `internal/ontostate.NextPhase(phase) (string, bool)` — the valid successor
  in the fixed order open → design → build → verify → close (returns ok=false at
  `close`, the terminal phase, and for unknown phases).
- Add a build-phase completion check `TasksAllChecked(tasksPath) (bool, error)` —
  true when `tasks.md` has at least one checkbox and no unchecked `- [ ]`.
- Add `onto advance <change>`: the gated phase-transition command. It runs the
  framework-install gate, loads the change's `onto-state.yaml`, derives the
  current phase, and advances to `NextPhase` ONLY IF the transition's
  precondition holds:
  - the artifacts required to ENTER the next phase already exist
    (`ValidateSkeleton`-style check against `RequiredArtifacts(next)`), and
  - leaving `build` additionally requires all `tasks.md` items checked.
  On success it writes the new phase via `ontostate.Save` and reports the
  transition; on a failed precondition it exits non-zero, names what is missing,
  and does NOT change the phase. Advancing past `close` is an error.
- **Dirty-worktree rule:** `onto advance` checks `git status --porcelain` in the
  workspace. A dirty worktree produces a WARNING for normal advances, but BLOCKS
  the release-critical `verify → close` transition (non-zero, no phase change).
- This change does NOT add dependency resolution or archive/close side effects
  beyond the phase write (#3c), nor `onto doctor` (#4) or packaging (#5).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains `onto advance` (valid-gate-only phase transitions with
  per-phase artifact preconditions, build-tasks-complete gate, and dirty-worktree
  blocking for close), plus per-phase `RequiredArtifacts` supersets and
  `NextPhase`/`TasksAllChecked` helpers.

## Impact

- `internal/ontostate`: per-phase `RequiredArtifacts`, `NextPhase`,
  `TasksAllChecked` (+ tests).
- `internal/ontocli`: new `advanceCmd()` (`onto advance`), registered on the
  root; a small git-porcelain helper (os/exec `git status --porcelain`) for the
  dirty-worktree check.
- No new dependency. `onto` stays isolated from homonto's projection pipeline
  (imports none of internal/{cli,engine,config,adapter,catalog}); shelling to
  `git` is allowed (it is the workflow's VCS, not the projection pipeline).
- Advances the onto workflow engine (#3b of 5 onto changes).

```

## openspec/changes/onto-phase-gates/design.md

- Source: openspec/changes/onto-phase-gates/design.md
- Lines: 1-101
- SHA256: 774959075c48f4d24c3212fb4c5cd007dc3a85de6ecc506ba8daad2fb7ab8a1c

```md
## Context

onto #1/#2/#3a are archived: the binary can create a change (`onto new`) and
validate its skeleton, but cannot advance it. #3b adds gated phase advancement —
the core of the onto workflow engine's transition enforcement.

## Goals / Non-Goals

**Goals:** per-phase `RequiredArtifacts` supersets; `NextPhase`;
`TasksAllChecked`; `onto advance` enforcing valid-gate-only transitions with
per-phase artifact preconditions + the build-tasks-complete gate + dirty-worktree
blocking for `verify → close`.

**Non-Goals:** dependency resolution + archive/close side effects (#3c); `onto
doctor` (#4); packaging (#5); auto-running skills; any change to homonto or the
isolation boundary.

## Decisions

**D1 — Per-phase cumulative `RequiredArtifacts` in `ontostate`.** Replace the
flat set with a cumulative map:
```
open   → onto-state.yaml, proposal.md, tasks.md
design → …, design.md
build  → …, plan.md
verify → …, verification.md
close  → same as verify
```
Unknown phase → the `open` base set. `ValidateSkeleton` (from #3a) is unchanged
in shape; it automatically tightens as a change advances. This is additive to the
`onto-binary` capability.

**D2 — `NextPhase` and `TasksAllChecked` in `ontostate`.**
`NextPhase(phase string) (string, bool)`: index into `["open","design","build",
"verify","close"]`; return the successor and true, or `("",false)` at `close`
and for unknown phases. `TasksAllChecked(tasksPath string) (bool, error)`: read
`tasks.md`; true iff it contains at least one checkbox (`- [ ]` or `- [x]`) and
no unchecked `- [ ]` (simple line scan, mirroring the comet checkoff idea).

**D3 — `advanceCmd()` = gate → load → next → precondition → dirty-check →
write.** `onto advance <change>` (positional arg + `--dir` default "."):
1. `gate(root)` (reuse from init.go); on error return it, no write.
2. `changeDir := <root>/docs/changes/<name>`; `st, err := ontostate.Load(<dir>/
   onto-state.yaml)`; error if missing/invalid (name-validate the arg too, reusing
   `validChangeName`).
3. `next, ok := NextPhase(st.Phase)`; if `!ok` → non-zero "already at terminal
   phase 'close'" (or "unknown phase"), no write.
4. Precondition: every `RequiredArtifacts(next)` file exists under `changeDir`
   (reuse a stat loop); if leaving `build` (i.e. `st.Phase=="build"`), also
   `TasksAllChecked(<dir>/tasks.md)` must be true. On failure → non-zero naming
   the missing artifact / incomplete tasks, no write.
5. Dirty-worktree: `worktreeDirty(root)` via `git status --porcelain`. If dirty:
   for `next=="close"` → non-zero "dirty worktree blocks close", NO write; else
   print a warning to stderr and continue.
6. Set `st.Phase = next`; `ontostate.Save(<dir>/onto-state.yaml, st)`; report
   `"<change>: <old> → <next>"`, exit 0.

**D4 — `worktreeDirty(root)` via os/exec git.** Run `git -C <root> status
--porcelain`; dirty iff output is non-empty. If git is absent or errors (not a
repo), treat as "cannot determine" → for `close` block conservatively with a
clear message; for normal advances, warn that cleanliness could not be verified
and continue. Shelling to `git` is allowed — it is the workflow's VCS, not the
projection pipeline; this does not break onto's isolation from homonto internal
packages.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontostate` | per-phase RequiredArtifacts, NextPhase, TasksAllChecked | os |
| `internal/ontocli` advance.go | `onto advance` (gate+precondition+dirty+write) | ontostate, os/exec, cobra |

`onto` still imports none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **git dependency for the dirty check** → `git` is universally present in an onto
  workflow (VCS-backed); the fallback (can't determine → block close, warn
  otherwise) is conservative and safe.
- **`TasksAllChecked` line-scan** → matches the comet/onto checkbox convention
  (`- [ ]`/`- [x]`); documented and tested; not a full markdown parse (YAGNI).
- **Advance is one step** → no multi-phase jump; keeps gates auditable. Fine.

## Testing Strategy

1. ontostate: RequiredArtifacts per phase (build needs plan.md, verify needs
   verification.md); NextPhase (each step + close→false + unknown→false);
   TasksAllChecked (all checked → true, one unchecked → false, none → false).
2. `onto advance` (temp workspaces, gate satisfied): open→design when design.md
   present; refuses when design.md missing (phase unchanged); build→verify blocked
   by an unchecked task; advance-past-close error; success writes the new phase
   (Load-back asserts it); name-validate + gate-failure paths.
3. Dirty-worktree: init a temp git repo, make it dirty; a normal advance warns but
   proceeds; `verify→close` is blocked (phase unchanged). Use a temp git repo via
   os/exec so the real repo is untouched.
4. Isolation grep; both binaries build; `go test [-race] ./...`, vet, gofmt, tidy.

## Open Questions

None blocking. Archive/close side effects (moving the change, syncing specs) and
dependency resolution are #3c.

```

## openspec/changes/onto-phase-gates/tasks.md

- Source: openspec/changes/onto-phase-gates/tasks.md
- Lines: 1-21
- SHA256: a7990e2b5ea28c886e472f8301a4de7d3ea0f1cbfde958f2f8ca11d18d29b415

```md
## 1. Phase helpers in `internal/ontostate`

- [ ] 1.1 (TDD, RED first) Make `RequiredArtifacts(phase)` cumulative per phase: open→[onto-state.yaml,proposal.md,tasks.md]; design→+design.md; build→+plan.md; verify/close→+verification.md; unknown→open base set. Tests: each phase's set; unknown → base. Confirm existing `ValidateSkeleton` tests still pass (a build-phase change missing plan.md now errors).
- [ ] 1.2 (TDD, RED first) `NextPhase(phase string) (string, bool)` over ["open","design","build","verify","close"]: successor+true; ("",false) at close and for unknown. Tests: each step, close→false, "bogus"→false.
- [ ] 1.3 (TDD, RED first) `TasksAllChecked(tasksPath string) (bool, error)`: read file; true iff ≥1 checkbox and no unchecked `- [ ]`. Tests: all `- [x]` → true; a mix with one `- [ ]` → false; no checkboxes → false; missing file → error.
- [ ] 1.4 GREEN; gofmt/vet clean for internal/ontostate.
- [ ] 1.5 Commit: `feat(ontostate): per-phase RequiredArtifacts + NextPhase + TasksAllChecked`

## 2. `onto advance` command (`internal/ontocli`)

- [ ] 2.1 (TDD, RED first) `worktreeDirty(root string) (dirty bool, determinable bool)` helper: run `git -C <root> status --porcelain` via os/exec; dirty iff non-empty output; determinable=false if git errors / not a repo. Test with a temp git repo (init, clean → not dirty; touch a file → dirty) — use os/exec so the real repo is untouched.
- [ ] 2.2 (TDD, RED first) `advanceCmd()` (positional `<change>` ExactArgs(1) + `--dir` default "."): gate(root) → validChangeName → Load `<dir>/docs/changes/<name>/onto-state.yaml` (error if missing/invalid) → `NextPhase` (error if terminal/unknown) → precondition: every `RequiredArtifacts(next)` file exists (stat loop, name the first missing) AND if leaving build, `TasksAllChecked(tasks.md)` (error if not) → dirty check: if `worktreeDirty` (or undeterminable) and next=="close" → non-zero, no write; else if dirty, warn to stderr and continue → set phase=next, `ontostate.Save`, report `"<change>: <old> → <next>"`, exit 0. Register `advanceCmd()` on the root.
- [ ] 2.3 (TDD, RED first) Tests via `NewRootCmd().SetArgs([]string{"advance",name,"--dir",tmp})` in a prepared workspace (gate satisfied; use a temp git repo so dirty state is controllable): open→design when design.md present (Load-back phase==design, exit 0); refuses when design.md missing (phase stays open, non-zero); create a build-phase change (write onto-state.yaml phase build + required files incl plan.md) with an unchecked task → build→verify refused (phase stays build); a change at close → advance errors, no write; a verify-phase change + dirty worktree → verify→close blocked (phase stays verify); a normal advance with a dirty worktree → warns but proceeds.
- [ ] 2.4 GREEN; `grep -E "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty; gofmt/vet clean.
- [ ] 2.5 Commit: `feat(onto): 'onto advance' gates phase transitions (+ dirty-worktree block on close)`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; E2E in a prepared temp git workspace: `onto new demo`, add `design.md`, `onto advance demo` → reports `open → design` and `onto status` shows phase design.
- [ ] 3.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3b (`onto advance` valid-gate transitions + dirty-worktree block) landed; remaining onto = dependency resolution + archive/close rules (#3c), `onto doctor` (#4), dual-binary packaging (#5). No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/onto-phase-gates/specs/onto-binary/spec.md

- Source: openspec/changes/onto-phase-gates/specs/onto-binary/spec.md
- Lines: 1-78
- SHA256: 3e4fac90a43aedb316c7b5005464e647aed1316641dcb5c8667798df6bc380ba

```md
## ADDED Requirements

### Requirement: Per-phase required artifacts

`internal/ontostate.RequiredArtifacts(phase)` SHALL return the cumulative set of
files that must exist at each workflow phase: `open` → `onto-state.yaml`,
`proposal.md`, `tasks.md`; `design` → those plus `design.md`; `build` → those
plus `plan.md`; `verify` and `close` → those plus `verification.md`. An unknown
phase SHALL return at least the `open` base set. `ValidateSkeleton` SHALL use this
per-phase set so a change's skeleton requirements tighten as it advances.

#### Scenario: build phase requires plan.md

- **GIVEN** a change at phase `build` with `onto-state.yaml`, `proposal.md`, `tasks.md`, `design.md` but no `plan.md`
- **WHEN** `ValidateSkeleton` runs
- **THEN** it returns an error naming `plan.md` as missing

### Requirement: onto advance gates phase transitions

`onto advance <change>` SHALL move a change to the next phase in the fixed order
`open → design → build → verify → close`, and ONLY through that order (no skips,
no reversals). It SHALL run the framework-install gate first. Before advancing it
SHALL verify the transition's precondition:

- the artifacts required to enter the NEXT phase (`RequiredArtifacts(next)`) all
  exist, AND
- when leaving `build`, every `tasks.md` checkbox is checked (at least one
  checkbox present, no unchecked `- [ ]`).

On success it SHALL write the new phase to `onto-state.yaml` and report the
transition. On a failed precondition it SHALL exit non-zero, name what is
missing, and leave the recorded phase unchanged. Advancing a change already at
`close` (or with an unknown phase) SHALL be an error with no write.

#### Scenario: advance open to design when design.md exists

- **GIVEN** a change at phase `open` with `design.md` present (and the open artifacts)
- **WHEN** `onto advance <change>` runs
- **THEN** the recorded phase becomes `design` and the command reports `open → design`, exiting 0

#### Scenario: advance refuses when the next phase's artifact is missing

- **GIVEN** a change at phase `open` with no `design.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero naming `design.md` as missing and the recorded phase stays `open`

#### Scenario: advance out of build requires all tasks checked

- **GIVEN** a change at phase `build` with `plan.md` present but an unchecked `- [ ]` item in `tasks.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating tasks are incomplete and the recorded phase stays `build`

#### Scenario: advance past close is an error

- **GIVEN** a change at phase `close`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating the change is already at the terminal phase and writes nothing

### Requirement: dirty worktree blocks the close transition

`onto advance` SHALL check the workspace's git worktree cleanliness (via `git
status --porcelain`). A dirty worktree SHALL produce a WARNING for a normal
transition (open→design, design→build, build→verify) but SHALL still allow it.
For the release-critical `verify → close` transition a dirty worktree SHALL BLOCK
the advance: the command exits non-zero, reports the dirty worktree, and does not
change the phase.

#### Scenario: dirty worktree warns but allows a normal advance

- **GIVEN** a change at phase `open` (with `design.md`) in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it advances to `design` (exit 0) after printing a dirty-worktree warning

#### Scenario: dirty worktree blocks verify to close

- **GIVEN** a change at phase `verify` (with `verification.md`) in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero reporting the dirty worktree and the recorded phase stays `verify`

```
