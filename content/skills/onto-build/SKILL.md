---
name: onto-build
description: onto phase 3 — plan and build. Use when an active change has phase build — writes the implementation plan, pauses at the plan-ready gate, then executes bite-sized tasks with one commit each under the chosen TDD/direct mode.
---

# onto-build — Phase 3: Plan and Build

Turn the confirmed design into a plan, then the plan into committed code —
one small, verified task at a time.

## Entry check

- `state.yaml` has `phase: build`.
- `workflow: full` → a `design.md` marked `Status: Confirmed` must exist; if
  it doesn't, the design phase isn't done — route back through `/onto`.
- Presets (`fix`/`tweak`) enter build directly after open-lite.
- On resume (fresh session, context loss): find the first unchecked task in
  `tasks.md`/`plan.md` and continue from there; never redo committed tasks.

## Steps

### 1. Write the plan

Write `docs/changes/<name>/plan.md`: bite-sized tasks mirroring `tasks.md`,
each with exact file paths, what to do, and how to verify it. A task that
can't state its verification isn't ready. Tasks should be small enough that
one commit each stays reviewable (~200 lines of change; split anything
bigger).

### 2. Plan-ready gate

> **GATE (plan-ready + execution config):** pause. The user reviews the plan
> and chooses the execution configuration, recorded in `state.yaml` under
> `decisions:`:
>
> - `isolation: branch | worktree` — branch for simple changes; worktree for
>   parallel work or a dirty current branch
> - `execution: direct | subagent` — direct in-session; subagent only when
>   real background dispatch capability exists
> - `tdd: tdd | direct` — tdd for anything with testable logic; direct for
>   content/docs deliverables
>
> This gate MAY be pre-authorized: if the user gave an explicit directive
> (e.g. "run to completion with defaults"), record it **verbatim** in
> `decisions.directive` and proceed with the recorded config — but still
> surface the plan summary so the user sees what will happen.

Create the isolation before the first task: `git checkout -b
<type>/YYYYMMDD/<change-name>` (or the worktree equivalent). Type prefix:
`feature` for full, `fix`/`tweak` for presets.

### 3. Execute task by task

For each task, in order:

1. **`tdd: tdd`** — write the failing test FIRST, run it, watch it fail for
   the expected reason; then write the minimal implementation; watch it
   pass. No production code without a failing test.
   **`tdd: direct`** — implement, then run the task's stated verification.
2. After verification passes: check the task off in `tasks.md` **and**
   `plan.md`, then commit — one commit per task, message reflects design
   intent. Never batch tasks into one commit; never leave checked-off tasks
   uncommitted.

### 4. Failure gate (systematic debugging)

On ANY build/test/unexpected failure: stop. Reproduce it, read the whole
error, check recent changes, trace the data flow, and identify the **root
cause**. No source fix may be proposed or applied before the root cause is
identified. If the root cause is a source bug, add a minimal failing test
that reproduces it, then fix, then watch the test pass. Symptom-patching is
prohibited.

### 5. Mid-build scope changes

- Small (missing edge case, scenario): edit the delta spec + design.md
  inline, append a task, note it in the commit message.
- Medium (interface/component/data-flow changes): pause, get user
  confirmation, revisit the design (back through the approach gate).
- Large (new capability, or new tasks exceed half the original task count):
  pause; the user chooses between splitting into a new change or expanding
  this one. Always fresh input.

## Exit checklist

- [ ] Every `tasks.md` item checked (or explicitly marked deferred-to-close
      with the reason) and every `plan.md` step done
- [ ] One commit per task; working tree clean
- [ ] Project build + test suite run fresh and pass (state the commands and
      results — do not rely on memory)
- [ ] `decisions:` in `state.yaml` filled (isolation, execution, tdd)
- [ ] `state.yaml` phase advanced: `build → verify`
- [ ] Announce the transition and load `onto-verify`
