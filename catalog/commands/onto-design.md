---
name: onto-design
description: Run the onto design phase — explore approaches and write design.md.
argument-hint: "The change to design (optional)"
---

# /onto-design

Run onto phase 2 (design): brainstorming-grade exploration, approach confirmation, then design.md plus ADR drafts and spec deltas. Load and follow the `onto-design` skill; if it is not installed, tell the user to
install the onto framework (declare `[frameworks.onto]`, then run `homonto
apply`) and stop. Every workflow state change goes through the `onto` binary —
never hand-edit `onto-state.yaml`.

`$ARGUMENTS`, if present, focuses this phase on the described work.
