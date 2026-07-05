# Tasks: doctor-opencode-link

## 1. Reproduce
- [ ] 1.1 Failing test: an owned skill with content + a correct Claude link but
      NO OpenCode link — `doctor` currently reports no OpenCode warning.

## 2. Fix
- [ ] 2.1 `engine.Doctor` checks both the Claude and OpenCode skill symlinks
      per owned skill, reporting `ok`/`warn` per tool.

## 3. Spec + regression
- [ ] 3.1 Delta `specs/cli-commands.md` — MODIFIED doctor requirement (both
      tools' links; drop "known gap").
- [ ] 3.2 Regression: full suite + existing doctor tests still pass.
