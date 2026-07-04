---
name: onto-verify
description: onto phase 4 — verify. Use when an active change has phase verify (all tasks checked) — picks a verification level from change scale, checks implementation against design and every spec scenario with fresh evidence, and writes verification.md.
---

# onto-verify — Phase 4: Verify

Prove — with fresh evidence, not recollection — that the implementation does
what the design and specs say. **Evidence before assertions, always.**

## Entry check

- `state.yaml` has `phase: verify` and every `tasks.md` item is checked
  (items explicitly marked deferred-to-close are allowed).
- Unchecked tasks mean build isn't done — the dispatcher's derivation table
  will send this back to build; route through `/onto`.

## Steps

### 1. Scale check → verification mode

Set `verify.mode` in `state.yaml`:

- **full** — `workflow: full`, any upgraded preset, a diff touching more
  than 5 files, or a new capability. Checks every delta-spec scenario, the
  full design, and the regression suite.
- **light** — a preset within its limits. Checks the changed behavior's
  scenarios plus the regression suite; the report may be brief but never
  absent.

### 2. Check against design and specs

For **every scenario in every delta spec** (workspace `specs/*.md`): run the
command(s) that demonstrate the behavior and capture the actual output.
Walk `design.md`'s key decisions and confirm the implementation matches —
deviations are findings, not footnotes. Re-run stated verifications from
`plan.md` where they are cheap.

Rules of evidence:

- Every claim needs a fresh command + its literal output. No "should work",
  no "passed earlier", no stale logs.
- A scenario that cannot be demonstrated is a **fail**, not a skip.

### 3. Regression

Run the project's full build and test suite. Capture the output.

### 4. Write the report

Write `docs/changes/<name>/verification.md`:

- Header: change, date, mode, git range (`base_ref..HEAD`)
- A table: requirement scenario → verdict (pass/fail) → evidence (literal
  command + output excerpt)
- Design-conformance notes and any deviations
- Regression results
- Final `verify.result: pass | fail` — mirror it into `state.yaml`

### 5. Failure gate

> **GATE (on any fail):** list the failing items and ask the user:
> **fix** (→ back to build: reset `phase: build`, add tasks for the fixes)
> or **accept deviation** (record each accepted deviation + rationale in
> `verification.md`; result becomes pass-with-deviations). Always fresh
> input — never auto-accept a failure. After three consecutive failed
> verify rounds, stop and make the user choose the path forward.

## Exit checklist

- [ ] `verification.md` exists with fresh evidence for every checked
      scenario, regression results included
- [ ] `verify.result: pass` (or pass-with-deviations, each recorded with
      rationale) in both the report and `state.yaml`
- [ ] `state.yaml` phase advanced: `verify → close`
- [ ] Announce the transition and load `onto-close`
