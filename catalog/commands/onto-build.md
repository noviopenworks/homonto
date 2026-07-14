---
name: onto-build
description: Run the onto build phase — plan, then execute tasks one commit each.
argument-hint: "The change to build (optional)"
---

# /onto-build

Run onto phase 3 (build): write the implementation plan, pause at the plan-ready gate, then execute bite-sized tasks with one commit each. Load and follow the `onto-build` skill; if it is not installed, tell the user to
install the onto framework (declare `[frameworks.onto]`, then run `homonto
apply`) and stop. Every workflow state change goes through the `onto` binary —
never hand-edit `onto-state.yaml`.

`$ARGUMENTS`, if present, focuses this phase on the described work.
