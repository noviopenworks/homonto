# tasks.md — canonical template

The change's checklist. Open creates the skeleton (boundaries); build
refines and checks items off — one commit per checked item.

## Template

```markdown
# Tasks: <change-name>

## 1. <area, e.g. Foundation>

- [ ] 1.1 <task — outcome-stated, small enough for one reviewable commit>
- [ ] 1.2 <task>

## 2. <area, e.g. Implementation>

- [ ] 2.1 <task>

## N. Validation

- [ ] N.1 <how this change proves itself — dry-runs, tests, evidence>
```

## Rules

- Checkbox syntax exactly `- [ ]` / `- [x]` (the phase-derivation table
  greps it). A deliberately deferred task uses `- [x] N.N DEFERRED to
  close: <reason>` — checked, with the deferral stated. Close is the only
  deferral target (build's exit and verify's entry recognize nothing
  else).
- Number tasks `<area>.<n>`; keep one outcome per task.
- Every change ends with a Validation area — a change that can't state its
  own proof isn't ready to build.
