---
name: onto-open
description: Open a new onto change — clarify scope and scaffold the workspace.
argument-hint: "What to build (optional)"
---

# /onto-open

Start onto phase 1 (open): clarify the requirement, check for scope splits, and create the change with `onto new`. Load and follow the `onto-open` skill; if it is not installed, tell the user to
install the onto framework (declare `[frameworks.onto]`, then run `homonto
apply`) and stop. Every workflow state change goes through the `onto` binary —
never hand-edit `onto-state.yaml`.

`$ARGUMENTS`, if present, focuses this phase on the described work.
