---
comet_change: onto-phase-gates
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-10-onto-phase-gates
status: final
---

# Onto Phase Gates — Technical Design

Refinement of `design.md` for `onto-phase-gates` (onto binary #3b). Adds gated
phase advancement: `onto advance` moves a change through open→design→build→
verify→close via valid gates only, with per-phase artifact preconditions, a
build-tasks-complete gate, and dirty-worktree blocking for close.

## Context

onto #1/#2/#3a archived: the binary creates a change (`onto new`) and validates
its skeleton, but cannot advance it. #3b is the transition-enforcement core of
the onto workflow engine. `onto` stays isolated from homonto's projection
pipeline; shelling to `git` (the workflow's VCS) is permitted.

## Goals / Non-Goals

**Goals:** per-phase `RequiredArtifacts` supersets; `NextPhase`;
`TasksAllChecked`; `onto advance` with valid-gate-only transitions + per-phase
preconditions + build-tasks gate + dirty-worktree block on close.

**Non-Goals:** dependency resolution + archive/close side effects (#3c); doctor
(#4); packaging (#5); auto-running skills; homonto/isolation changes.

## Decisions

**D1 — Cumulative per-phase `RequiredArtifacts` in `ontostate`.**
```
open→[onto-state.yaml,proposal.md,tasks.md]; design→+design.md;
build→+plan.md; verify,close→+verification.md; unknown→open base
```
`ValidateSkeleton` (from #3a) is unchanged in shape and tightens automatically.

**D2 — `NextPhase`, `TasksAllChecked` in `ontostate`.**
`NextPhase(phase) (string,bool)` over the fixed order; `("",false)` at close and
unknown. `TasksAllChecked(path) (bool,error)`: line-scan `tasks.md`; true iff ≥1
`- [ ]`/`- [x]` checkbox and zero unchecked `- [ ]`. Not a full markdown parse
(YAGNI); matches the comet/onto checkbox convention.

**D3 — `advanceCmd()`: gate → load → next → precondition → dirty-check → write.**
`onto advance <change>` (ExactArgs(1) + `--dir` default "."):
1. `gate(root)` (reuse init.go); error → no write.
2. `validChangeName(name)` (reuse); Load `<dir>/docs/changes/<name>/onto-state.yaml`
   (error if missing/invalid).
3. `next,ok := NextPhase(st.Phase)`; `!ok` → non-zero "already at terminal
   phase 'close'" / "unknown phase", no write.
4. Precondition — the CURRENT phase's deliverables gate LEAVING it (a phase's
   artifacts are produced while the change is IN that phase): stat every
   `RequiredArtifacts(st.Phase)` under changeDir (name first missing); AND if
   `st.Phase=="build"`, `TasksAllChecked(<dir>/tasks.md)` must be true (else
   "tasks incomplete"). Fail → non-zero, no write. (Leaving `open` needs only
   proposal.md + tasks.md; leaving `design` needs `design.md`; etc. — NOT
   `RequiredArtifacts(next)`, which would require a phase's output before it runs.)
5. `dirty, ok := worktreeDirty(root)`. If `next=="close"` and (dirty OR !ok) →
   non-zero "dirty worktree blocks close" / "cannot verify worktree", no write.
   Else if dirty → warn to stderr, continue.
6. `st.Phase = next`; `ontostate.Save`; report `"<change>: <old> → <next>"`,
   exit 0.

**D4 — `worktreeDirty(root) (dirty, determinable bool)` via os/exec git.**
`exec.Command("git","-C",root,"status","--porcelain")`; dirty iff stdout
non-empty; determinable=false if git errors / not a repo. Conservative fallback:
undeterminable blocks `close` (with a clear "could not verify worktree" message)
and only warns for normal advances. Shelling to git does not couple `onto` to any
homonto internal package.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontostate` | per-phase RequiredArtifacts, NextPhase, TasksAllChecked | os |
| `internal/ontocli` advance.go | `onto advance` (gate+precondition+dirty+write) | ontostate, os/exec, cobra |

`onto` imports none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **git for the dirty check** → universal in a VCS-backed onto workflow; the
  undeterminable→block-close-warn-otherwise fallback is safe.
- **`TasksAllChecked` line-scan** → convention-based, tested; not a markdown
  parser (YAGNI).
- **One-step advance** → keeps gates auditable; no multi-phase jumps.
- **Existing ValidateSkeleton tests tighten** → a build-phase fixture now needs
  plan.md; update those fixtures as part of Task 1 (they are this change's tests).

## Testing Strategy

1. ontostate: per-phase RequiredArtifacts; NextPhase (each step, close→false,
   unknown→false); TasksAllChecked (all-checked/one-unchecked/none/missing).
2. `onto advance` over temp workspaces (gate satisfied, temp git repo for dirty
   control): open→design ok / missing-design refused / build→verify blocked by
   unchecked task / past-close error / success writes phase (Load-back) /
   name+gate failure paths.
3. Dirty-worktree: temp git repo made dirty → normal advance warns+proceeds;
   verify→close blocked (phase unchanged). Real repo untouched (all in t.TempDir).
4. Isolation grep; both binaries build; `go test [-race] ./...`, vet, gofmt, tidy.

## Open Questions

None blocking. Archive/close side effects + dependency resolution → #3c.
