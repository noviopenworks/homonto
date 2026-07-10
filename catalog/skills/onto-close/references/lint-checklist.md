# Close-phase lint checklist

Agent-run checks (grep/read — no scripts), staged: sections run at the
points onto-close names (§1–2 before the merge, §3 after it, §4 before
archiving). Findings block the archive step exactly like the guides
obligation: fix them or stop.

## 1. Delta spec format (each workspace `specs/<capability>.md`)

- [ ] Section headings are only `## ADDED Requirements`,
      `## MODIFIED Requirements`, `## REMOVED Requirements`,
      `## RENAMED Requirements` — nothing else, empty sections omitted
- [ ] Every `### Requirement:` block **in ADDED/MODIFIED sections** has
      SHALL or MUST in its first non-empty line after the heading
      (REMOVED bodies are removal rationales; RENAMED has no bodies —
      neither is subject to this rule)
- [ ] **Every** `#### Scenario:` block has WHEN and THEN bullets (GIVEN
      optional), and each ADDED/MODIFIED requirement has ≥1 scenario
- [ ] MODIFIED/REMOVED/RENAMED names match the living spec **exactly**
      (grep the living file for each name) — except a MODIFIED name may
      instead match the TO name of a RENAMED entry in the same delta
- [ ] A MODIFIED/REMOVED/RENAMED section in a delta whose capability has
      **no living spec file** is itself a finding
- [ ] RENAMED entries are `- FROM:` / `  TO:` pairs

## 2. Workspace state

- [ ] `state.yaml` parses as YAML; enum fields hold allowed values and
      typed fields hold their types (`deps` a list, `metrics.phases` a
      map, counters numeric — schema: `onto/references/state-yaml.md`)
- [ ] `verification.md` has a current `Result:` line
- [ ] Every ADR draft has `**Status:**`, `**Date:**`, `**Change:**` fields
- [ ] Every template-based artifact **that exists** follows its
      template's section structure — proposal, design, tasks, notes,
      plan, verification — checked against their references (deviation
      anywhere is a finding; presets legitimately lack design/plan and
      possibly notes)

## 3. Post-merge (run AFTER the spec merge, before archive)

- [ ] Living specs contain **no** delta-only section headings:
      `grep -nE '^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements' docs/specs/*.md`
      → no matches outside `docs/specs/README.md` (the README legitimately
      documents the section names; prose mentions anywhere are fine — the
      check is heading-anchored)
- [ ] Merged requirements read as current truth — no change-log language
- [ ] Scenario structure intact in every touched living spec

## 4. Pre-archive

- [ ] `guides` is not `pending` (resolved in the guides-obligation step —
      checked here because it cannot be satisfied before that step runs)
- [ ] No unresolved `DEFERRED to close:` markers remain in `tasks.md` —
      deferred tasks are executed during close, before the final
      confirmation; archiving one undone is prohibited
- [ ] No live doc (README, docs/guides, docs/specs, docs/adr, skills)
      references a path this change moved or deleted — archives are exempt
      (history may cite old paths)
