# Plan: polish-onto-framework

Design: `design.md` (Status: Confirmed 2026-07-04). One commit per task.

## Task 1 — dispatcher: state template, deps, degrade rule

- [x] done
- Files: content/skills/onto/references/state-yaml.md (new),
  content/skills/onto/SKILL.md, docs/changes/README.md
- Do: canonical state schema/template/rebuild rules; deps awareness in
  discovery; degrade-don't-halt rule; contracts point to the template
- Verify: derivation tables byte-identical; deps text matches delta spec

## Task 2 — onto-open: templates + notes checkpoint

- [x] done
- Files: content/skills/onto-open/references/{proposal,tasks,notes}.md
  (new), content/skills/onto-open/SKILL.md
- Do: canonical templates (Preset:/Depends-on: markers, checkbox rules,
  checkpoint pattern); SKILL creates artifacts from templates, seeds and
  maintains notes.md, stamps metrics.phases.open
- Verify: template sections match SKILL assumptions (self-application)

## Task 3 — onto-design: templates + checkpoint + parallel exploration

- [x] done
- Files: content/skills/onto-design/references/{design,adr-draft,delta-spec}.md
  (new), content/skills/onto-design/SKILL.md
- Do: templates (Status lines, ADR fields, delta sections incl. RENAMED);
  notes.md read/update protocol; optional parallel approach exploration;
  stamp metrics.phases.design
- Verify: delta template encodes the lint rules

## Task 4 — onto-build: plan template + subagent protocol

- [x] done
- Files: content/skills/onto-build/references/{plan,subagent-protocol}.md
  (new), content/skills/onto-build/SKILL.md
- Do: plan template with done markers + risk marker; coordinator/worker
  protocol (dispatch contents, repo-verified returns, reviewer rule,
  serial execution); stamp metrics.phases.build
- Verify: protocol text matches delta spec ("coordinator never implements")

## Task 5 — onto-verify: verification template + adversarial protocol

- [x] done
- Files: content/skills/onto-verify/references/{verification,adversarial}.md
  (new), content/skills/onto-verify/SKILL.md
- Do: report template with Result: rule; two-skeptic protocol
  (refute-not-approve, triage, skip recording); verify_rounds increment;
  stamp metrics.phases.verify
- Verify: Result rule present; skeptic prompts forbid approval

## Task 6 — onto-close: lint + RENAMED + metrics + ship

- [x] done
- Files: content/skills/onto-close/references/{lint-checklist,ship-handoff}.md
  (new), content/skills/onto-close/SKILL.md, docs/specs/README.md,
  docs/adr/README.md
- Do: staged lint (blocking); RENAMED→MODIFIED→REMOVED→ADDED merge order;
  metrics finalization; ship-handoff offer; contracts updated
- Verify: lint covers every delta-spec lint scenario

## Task 7 — presets: reuse + metrics

- [x] done
- Files: content/skills/onto-fix/SKILL.md, content/skills/onto-tweak/SKILL.md
- Do: explicit reference reuse pointers; notes.md recommendation; metrics
  stamps; durable upgrade annotation
- Verify: no duplicated template content in presets

## Task 8 — guide

- [x] done
- Files: docs/guides/onto-workflow.md
- Do: templates/references, subagent execution, adversarial verify,
  checkpoints/recovery, deps/parallel changes, ship handoff, metrics
- Verify: every named reference file exists; links resolve

## Task 9 — validation (risk: high)

- [x] done
- Files: docs/changes/polish-onto-framework/validation-notes.md
- Do: two fresh-context dry-run agents (lifecycle; lint/adversarial/deps);
  fix all findings; mechanical checks (containment, table identity,
  template conformance, go test)
- Verify: dry-run reports recorded; all defects fixed; checks green
