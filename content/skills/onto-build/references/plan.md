# plan.md — canonical template

The executable breakdown of the confirmed design. A task that can't state
its verification isn't ready.

## Template

```markdown
# Plan: <change-name>

Design: `design.md` (Status: Confirmed <date>). One commit per task.

## Task N — <outcome, imperative>  <!-- add `(risk: high)` when it warrants a reviewer -->

- Files: <exact paths created/modified>
- Do: <what, concretely — reference design sections, don't restate them>
- Verify: <the command(s)/check(s) that prove this task done>
```

## Rules

- Bite-sized: one reviewable commit (~200 lines of change) per task —
  split anything bigger.
- `(risk: high)` marks tasks that get a reviewer agent under
  `execution: subagent` (and deserve extra scrutiny under `direct`).
- Tasks map to `tasks.md` areas; when a plan task completes, check BOTH
  files.
- The final task is always validation (the change proving itself).
