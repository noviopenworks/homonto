---
name: to-plan
description: Run the to plan phase — write plan.md as bite-sized, verifiable tasks.
argument-hint: "The change to plan (optional)"
---

# /to-plan

Run to phase 1 (plan): write `docs/tasks/<name>/plan.md` — a short goal
statement plus a checklist of bite-sized tasks, each naming its files and its
verification command — then advance with `to phase <name>`. Load and follow
the `to-plan` skill; if it is not installed, tell the user to install the to
framework (declare `[frameworks.to]`, then run `homonto apply`) and stop.
Every workflow state change goes through the `to` binary — never hand-edit
`to-state.yaml`.

`$ARGUMENTS`, if present, focuses this phase on the described work.
