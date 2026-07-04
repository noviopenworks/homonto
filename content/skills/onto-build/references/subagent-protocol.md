# Subagent build protocol (`execution: subagent`)

Coordinator/worker execution for the build phase. **The main session NEVER
implements** — it plans, dispatches, verifies, and keeps state true.

## When to choose subagent over direct

- Many independent tasks (≳4) or tasks touching disjoint files
- Main-session context is precious (long-running change, big design)
- Tasks benefit from fresh eyes (no accumulated assumptions)

`direct` remains right for small serial changes where dispatch overhead
exceeds the work.

## Per-task dispatch

For each unchecked task, in plan order (parallelize only tasks with
disjoint files), dispatch ONE fresh-context implementer agent whose prompt
contains:

1. The task text verbatim (files, do, verify) from `plan.md`
2. The relevant `design.md` section(s) — pasted, not summarized
3. Conventions: one commit for this task, message style from recent
   `git log`, match surrounding code idiom
4. The TDD rule in force (`tdd: tdd` → failing test first, watch it fail)
5. The debugging rule: on any failure, root cause before any fix —
   reproduce, read the whole error, trace; no symptom-patching
6. The bookkeeping obligation: after verification passes, check the task
   off in BOTH `tasks.md` and `plan.md`, then commit — files, not chat
7. Return contract: diff summary + literal verification output

## Coordinator duties after each return

- **Verify against the repository, not the report**: the commit exists
  (`git log`), the checkoffs landed in both files, the working tree is
  clean, and the stated verification output is plausible (spot-run it
  when cheap).
- A failed or half-done task is re-dispatched with the failure context, or
  taken through the failure gate — never silently absorbed.

## Reviewer agents

After any task marked `(risk: high)` — and always after the final task —
dispatch a fresh reviewer agent with the diff range and the design
section, prompted to **find faults** (correctness, spec conformance,
missed edge cases), never to approve. CRITICAL findings are fixed before
the next task; accepted non-critical findings are recorded in the plan or
commit body.

## Failure of the protocol itself

No real dispatch capability available → record the fact, fall back to
`execution: direct` in `state.yaml`, announce, continue.
