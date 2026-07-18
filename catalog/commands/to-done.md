---
name: to-done
description: Run the to done phase — real verification, one skeptic pass, then `to done --verified` archives the change.
argument-hint: "The change to finish (optional)"
---

# /to-done

Run to phase 3 (done): run the plan's verify command, dispatch `to-skeptic`
exactly once (sequentially — never in parallel), triage its findings, record
the outcome in `plan.md`, then finish with `to done <name> --verified`. The
`--verified` flag is self-asserted by design — this skill is what makes the
assertion true before it is made; never pass it early. Load and follow the
`to-done` skill; if it is not installed, tell the user to install the to
framework (declare `[frameworks.to]`, then run `homonto apply`) and stop.

`$ARGUMENTS`, if present, focuses this phase on the described work.
