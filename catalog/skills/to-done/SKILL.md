---
name: to-done
description: to phase 3 — done. Use when a change's plan is fully executed — runs the real verification, dispatches the single skeptic pass, records the outcome, then `to done --verified` archives the change.
---

# to-done — Phase 3: Done

Finish honestly. The binary will accept `--verified` from anyone; this skill
is what makes the assertion true before it is made.

## Entry check

- `to status --json` shows the change at `phase: do` with every plan task
  checked.
- The working tree is committed (the binary is git-blind and won't check —
  you do).

## Steps

1. **Run the plan's verify command** — the bottom-line one, not just the
   per-task checks — and read the output.
2. **Dispatch `to-skeptic`, exactly once.** Hand it the change notes, the
   plan, and the claim being made ("this change works because …"). One
   skeptic, one pass, sequential — to never dispatches subagents in parallel,
   and there is no second lens coming; its single pass covers both the claims
   and the gaps.
3. **Triage its findings.** Fix what's real (back through the `to-do` loop for
   anything substantial), decline the rest with a written reason. Re-run the
   verify command after any fix.
4. **Record the outcome** at the bottom of `plan.md`: the literal verify
   command, its result, and the skeptic's verdict (including declined
   findings). De-slop the prose. Commit.
5. **Assert and archive:**
   `to done <name> --verified --evidence "<the literal verify command and its result>"`.
   The evidence string is recorded verbatim in the archived state — it is
   what makes this verification distinguishable from a skipped one later.
   The change moves to `docs/tasks/archive/<date>-<name>/`.

## Rules

- **Never pass `--verified` before steps 1–4 are done.** The checkbox is
  self-asserted by design; asserting it without the work is lying in writing,
  in a reviewable artifact.
- If verification fails and the fix isn't obvious, stay in `do` — tell the
  user rather than force the finish.
- Never hand-edit `to-state.yaml`; archiving is the binary's move, not `mv`.
