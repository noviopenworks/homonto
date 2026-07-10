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
