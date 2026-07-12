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
- Read `notes.md` at entry when present — accepted decisions and recorded
  directives inform what to verify against.
- Unchecked tasks mean build isn't done — the dispatcher's derivation table
  will send this back to build; route through `/onto`.

## Steps

### 1. Scale check → verification mode

Set `verify.mode` in `state.yaml`:

- **full** — `workflow: full`, any upgraded preset, a diff touching more
  than 5 files in `base_ref..HEAD`, or a new capability. Checks every
  delta-spec scenario, the full design, and the regression suite.
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

### 2b. Adversarial pass

After the self-evidence table is drafted, follow
`references/adversarial.md`: **full mode requires two parallel
fresh-context skeptics** — conformance (refute each scenario claim) and
robustness (edge cases, drift/recovery paths) — prompted to refute, never
approve; light mode uses one optional skeptic with skips recorded. Triage
findings per the protocol: a refuted claim fails its scenario; new defects
are CRITICAL-fix or gate-decided deviations. No dispatch capability →
record the skipped pass in the report's Adversarial section
(protocol-mandated skips live there, no acceptor needed). Increment
`metrics.verify_rounds` once per round.

### 3. Regression

Run the project's full build and test suite. Capture the output. If the
project has no build/test suite (e.g. a content-only repo), record that
fact as the regression result — that is a valid result, not a skipped
check.

### 4. Write the report

Write `docs/changes/<name>/verification.md` from the canonical template
`references/verification.md` (header with machine-read `Result:` line,
scenario-evidence table, design conformance, adversarial pass, regression,
deviations). Mirror the result into `state.yaml` `verify.result`.

### 5. Failure gate

> **GATE (on any fail):** list the failing items and ask the user:
> **fix** (→ back to build: reset `phase: build`, add tasks for the fixes)
> or **accept deviation** (record each accepted deviation + its rationale
> in `verification.md`; the `Result:` line and `verify.result` stay `pass`
> — deviations live in the report, not in the enum). Always fresh input —
> never auto-accept a failure. After three consecutive failed verify
> rounds, stop and make the user choose the path forward.

## Exit checklist

- [ ] `verification.md` exists with a `Result:` line and fresh evidence for
      every checked scenario, regression results included
- [ ] `verify.result: pass` in both the report and `state.yaml` (accepted
      deviations, if any, each recorded with rationale in the report)
- [ ] Adversarial pass run (or its skip recorded in the report's
      Adversarial section); `metrics.verify_rounds` incremented
- [ ] onto-no-slop pass run over `verification.md`
- [ ] `state.yaml` phase advanced: `verify → close`;
      `metrics.phases.verify: <today>` stamped
- [ ] Announce the transition and load `onto-close`
