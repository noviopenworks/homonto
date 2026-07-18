---
name: to-do
description: to phase 2 — do. Use when an active change has phase do — executes plan.md one task at a time through the implementer/reviewer loop, strictly sequentially, holding the code to onto-grade standards.
---

# to-do — Phase 2: Do

Execute the plan, one task at a time. This is the code-writing skill: the flow
is simple, but the code written inside it is held to the full bar.

## Entry check

- `to status --json` shows the change at `phase: do`.
- `plan.md` has a task list. If it doesn't, the plan phase isn't done — route
  back through `/to`.
- On resume (fresh session, context loss): run `to handoff <name>` first, then
  find the first unchecked task in `plan.md` and continue from there; never
  redo completed tasks.

## The loop

For each unchecked task in `plan.md`, in order:

1. **Dispatch `to-implementer`** with the task verbatim: the files to touch,
   what to change, and the task's verification command. One implementer at a
   time — **never in parallel**.
2. **Verify against the repository**, not the report: check the diff exists
   and the task's verification command passes.
3. **Dispatch `to-reviewer`** on the diff. One reviewer, after the
   implementer returns — never alongside it.
4. **Act on findings.** Fix critical/major findings before moving on
   (re-dispatch the implementer for substantial fixes; apply trivial ones
   directly). Declined findings are recorded with a reason in the change
   notes — never silently dropped.
5. **Check off the task** in `plan.md` and commit: one task, one commit, a
   message that names the task. De-slop the message.

Small tasks (a rename, a doc line) may skip the subagent loop and be done
directly — but never skip the verification command or the commit.

## Code-writing standards (every task, no exceptions)

- Read the surrounding code before changing it; match its style, naming,
  idioms, and comment density.
- The smallest change that satisfies the task; no unrelated refactors.
- Behavior changes get focused tests, added or updated in the same task.
- Run the narrowest useful verification and read its output; a green run you
  didn't read is not verification.
- No symptom patches: when something fails unexpectedly, find the root cause
  before changing anything.

## Exit

All tasks checked and the plan's bottom-line verify command passing → load the
`to-done` skill. Do not run `to done` from here; finishing is `to-done`'s job.
If the work is not worth finishing, `to abandon <name>` and tell the user why.
