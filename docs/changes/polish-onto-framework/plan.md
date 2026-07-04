# Plan: polish-onto-framework

All tasks complete (done markers below per task).

Design: `design.md` (Status: Confirmed 2026-07-04). One commit per task.
Verification per task = the stated checks; regression `go test ./...` at
the end (no Go changes expected).

## Task 1 — dispatcher: state template, deps, degrade rule

- [x] done

- Create `content/skills/onto/references/state-yaml.md`: full schema
  (incl. `deps`, `metrics`, `decisions.directive`) + fenced template +
  per-field rebuild rules (aligned with drift-detection vocabulary).
- Update `docs/changes/README.md`: add deps/metrics to schema block,
  point to the template as canonical, add per-field rebuild rules for the
  new fields.
- Update `content/skills/onto/SKILL.md`: discovery table shows deps
  status; blocked-deps warning (proceed/switch/stop); references pointer +
  degrade-don't-halt fallback line; routing note for worktree-per-change.
- Verify: derivation tables still byte-identical; deps scenario text
  matches delta spec.

## Task 2 — onto-open: templates + notes checkpoint

- [x] done

- Create `content/skills/onto-open/references/{proposal.md,tasks.md,notes.md}`
  (canonical templates with per-section rules; proposal gains optional
  `Depends-on:` line).
- Update SKILL.md: create artifacts from templates; create notes.md at
  open; update notes.md before ending any decision-producing turn; read
  notes.md at entry; stamp `metrics.phases.open` at exit.
- Verify: template sections match what this change's own proposal used
  (self-application: adjust template, not history).

## Task 3 — onto-design: templates + checkpoint + parallel exploration

- [x] done

- Create `content/skills/onto-design/references/{design.md,adr-draft.md,delta-spec.md}`.
- Update SKILL.md: notes.md read/update protocol; OPTIONAL parallel
  approach exploration (2–3 fresh agents sketch approaches when genuinely
  open, main session synthesizes — MAY, not MUST); templates; stamp
  `metrics.phases.design`.
- Verify: delta-spec template encodes the lint rules (SHALL first line,
  scenario shape, four section kinds incl. RENAMED).

## Task 4 — onto-build: plan template + subagent protocol

- [x] done

- Create `content/skills/onto-build/references/{plan.md,subagent-protocol.md}`
  (protocol: coordinator/worker roles, per-task dispatch contents,
  file-based checkoff verification, reviewer-after-risk rule, when to
  choose subagent vs direct).
- Update SKILL.md: execution branch — `direct` unchanged; `subagent` →
  follow the protocol reference; plan template usage; `risk:` marker for
  tasks; stamp `metrics.phases.build`.
- Verify: protocol scenario text matches delta spec ("coordinator never
  implements").

## Task 5 — onto-verify: verification template + adversarial protocol

- [x] done

- Create `content/skills/onto-verify/references/{verification.md,adversarial.md}`
  (skeptic prompts: conformance + robustness, refute-not-approve, triage
  rules, no-capability deviation rule).
- Update SKILL.md: adversarial step after self-evidence (full: required 2
  skeptics; light: optional 1, skip recorded); `verify_rounds` increment;
  stamp `metrics.phases.verify`.
- Verify: template has the `Result:` line rule; skeptic prompts forbid
  approval language.

## Task 6 — onto-close: lint + RENAMED + metrics + ship

- [x] done

- Create `content/skills/onto-close/references/{lint-checklist.md,ship-handoff.md}`.
- Update SKILL.md: lint step before merge (findings block like guides);
  RENAMED merge semantics; metrics finalization (tasks_total,
  verify_rounds, upgraded, phase dates); ship-handoff offer after archive.
- Update `docs/specs/README.md`: RENAMED section semantics.
- Update `docs/adr/README.md`: point template to onto-design reference.
- Verify: lint checklist covers every delta-spec lint scenario.

## Task 7 — presets: reuse + metrics

- [x] done

- Update `content/skills/onto-fix/SKILL.md` + `onto-tweak/SKILL.md`:
  reuse open/build/verify/close references (explicit pointers), notes.md
  optional-but-recommended for presets, metrics stamps, `upgraded: true`
  on upgrade.
- Verify: no duplicated template content in presets.

## Task 8 — guide

- [x] done

- Update `docs/guides/onto-workflow.md`: templates & references section,
  subagent execution, adversarial verify, notes.md, deps/parallel work,
  ship handoff, metrics. Keep it a guide — link, don't duplicate.
- Verify: every named reference file exists; links resolve.

## Task 9 — validation (risk: high)

- [x] done

- Dry-run A (fresh agent): full lifecycle with templates + notes.md +
  subagent-protocol simulation; report defects.
- Dry-run B (fresh agent): adversarial-verify simulation + close lint fed
  a deliberately malformed delta (missing SHALL, broken scenario, bad
  RENAMED) — must catch all; deps warning; metrics stamping.
- Fix everything they find; record in validation-notes.md.
- Self-containment grep over content/skills/ incl. references/;
  derivation-table byte-identity; template-conformance check of this
  change's own workspace; `go test ./...`.
