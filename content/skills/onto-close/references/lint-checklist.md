# Close-phase lint checklist

Agent-run checks (grep/read — no scripts) executed BEFORE merging.
Findings block the archive step exactly like the guides obligation: fix
them or stop.

## 1. Delta spec format (each workspace `specs/<capability>.md`)

- [ ] Section headings are only `## ADDED Requirements`,
      `## MODIFIED Requirements`, `## REMOVED Requirements`,
      `## RENAMED Requirements` — nothing else, empty sections omitted
- [ ] Every `### Requirement:` block's **first line** contains SHALL or
      MUST
- [ ] Every ADDED/MODIFIED requirement has ≥1 `#### Scenario:` with
      GIVEN/WHEN/THEN bullets
- [ ] MODIFIED/REMOVED/RENAMED names match the living spec **exactly**
      (grep the living file for each name)
- [ ] RENAMED entries are `- FROM:` / `  TO:` pairs

## 2. Workspace state

- [ ] `state.yaml` parses as YAML; enum fields hold allowed values
      (schema: `onto/references/state-yaml.md`); `guides` is not `pending`
- [ ] `verification.md` has a current `Result:` line
- [ ] Every ADR draft has `**Status:**`, `**Date:**`, `**Change:**` fields
- [ ] Artifacts follow their templates' section structure (spot-check
      proposal/design/tasks against the references)

## 3. Post-merge (run AFTER the spec merge, before archive)

- [ ] Living specs contain **no** delta-only headings
      (`grep -n "ADDED\|MODIFIED\|REMOVED\|RENAMED" docs/specs/*.md` → none)
- [ ] Merged requirements read as current truth — no change-log language
- [ ] Scenario structure intact in every touched living spec

## 4. Dangling references

- [ ] No live doc (README, docs/guides, docs/specs, docs/adr, skills)
      references a path this change moved or deleted — archives are exempt
      (history may cite old paths)
