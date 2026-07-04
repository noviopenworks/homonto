# Tasks: add-onto-workflow

## 1. Foundation

- [x] 1.1 Create `docs/` workflow layout skeleton (`adr/`, `specs/`,
      `changes/`, `changes/archive/`, `guides/`) with a README in each
      explaining its contract
- [x] 1.2 Define the `state.yaml` schema and document it (fields, lifecycle,
      file-state-wins recovery rule)
- [x] 1.3 Define the ADR template and numbering convention

## 2. Skill Set

- [x] 2.1 Author `content/skills/onto/SKILL.md` — dispatcher: phase detection
      from state.yaml + file cross-check, routing table, resume rules,
      rtk/graphify preflight, GitHub entry-point contract
- [x] 2.2 Author `content/skills/onto-open/SKILL.md` — clarification,
      split preflight, proposal/design/tasks creation, review blocking point
- [x] 2.3 Author `content/skills/onto-design/SKILL.md` — brainstorming-grade
      design, Design Doc, ADR drafts, spec deltas, approach confirmation
- [ ] 2.4 Author `content/skills/onto-build/SKILL.md` — implementation plan,
      plan-ready pause, TDD/direct modes, commit-per-task, failure handling
- [ ] 2.5 Author `content/skills/onto-verify/SKILL.md` — verification levels,
      checks vs design/specs/tasks, verification.md, fail decision point
- [ ] 2.6 Author `content/skills/onto-close/SKILL.md` — spec delta merge, ADR
      status finalization, docs/guides obligation, archive, final confirmation
- [ ] 2.7 Author `content/skills/onto-fix/SKILL.md` and
      `content/skills/onto-tweak/SKILL.md` — preset paths + upgrade rules

## 3. Integration

- [ ] 3.1 Wire onto skills into `homonto.toml` `[skills]` and run
      `homonto apply` to symlink into `.claude/skills/` (dogfood proof)
- [ ] 3.2 Document GitHub entry points (resolve-issue / continue-pr → onto)
      in the dispatcher skill and a `docs/guides/onto-workflow.md` guide
- [ ] 3.3 Update `README.md` with the development-workflow section

## 4. Migration

- [ ] 4.1 Migrate `openspec/specs/*` → `docs/specs/` (flatten to
      `<capability>.md`)
- [ ] 4.2 Migrate archived change + `docs/superpowers/` artifacts →
      `docs/changes/archive/` and extract ADR-worthy decisions → `docs/adr/`
- [ ] 4.3 Retire `openspec/` and `docs/superpowers/` (this change's own
      workspace moves last, at close)

## 5. Validation

- [ ] 5.1 Dry-run walkthrough: simulate a full onto lifecycle on a scratch
      change (open→close) verifying each blocking point, state transition,
      and artifact contract
- [ ] 5.2 Dry-run preset paths: /onto-fix and /onto-tweak including an
      upgrade-rule trigger
- [ ] 5.3 Verify skills load via symlink (`/onto` visible to Claude Code) and
      contain no references to openspec CLI or comet scripts
