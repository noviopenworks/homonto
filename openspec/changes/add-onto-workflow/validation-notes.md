# Validation Notes: add-onto-workflow

## Dogfood (task 3.1, evidence captured 2026-07-04)

Two product bugs were found and fixed first (task 3.4, commit d93187e):
skills-only configs never applied links; relative content dir made
dangling link targets. Evidence below is from the fixed binary.

### `./homonto plan` (before apply)
```
claude:
  + skill.onto-verify = /home/mg/.claude/skills/onto-verify -> /home/mg/homonto/content/skills/onto-verify
  + skill.onto-close = /home/mg/.claude/skills/onto-close -> /home/mg/homonto/content/skills/onto-close
  + skill.onto-fix = /home/mg/.claude/skills/onto-fix -> /home/mg/homonto/content/skills/onto-fix
  + skill.onto-tweak = /home/mg/.claude/skills/onto-tweak -> /home/mg/homonto/content/skills/onto-tweak
  + skill.onto = /home/mg/.claude/skills/onto -> /home/mg/homonto/content/skills/onto
  + skill.onto-open = /home/mg/.claude/skills/onto-open -> /home/mg/homonto/content/skills/onto-open
  + skill.onto-design = /home/mg/.claude/skills/onto-design -> /home/mg/homonto/content/skills/onto-design
  + skill.onto-build = /home/mg/.claude/skills/onto-build -> /home/mg/homonto/content/skills/onto-build
opencode:
  + skill.onto-build = /home/mg/.config/opencode/skills/onto-build -> /home/mg/homonto/content/skills/onto-build
  + skill.onto-verify = /home/mg/.config/opencode/skills/onto-verify -> /home/mg/homonto/content/skills/onto-verify
  + skill.onto-close = /home/mg/.config/opencode/skills/onto-close -> /home/mg/homonto/content/skills/onto-close
  + skill.onto-fix = /home/mg/.config/opencode/skills/onto-fix -> /home/mg/homonto/content/skills/onto-fix
  + skill.onto-tweak = /home/mg/.config/opencode/skills/onto-tweak -> /home/mg/homonto/content/skills/onto-tweak
  + skill.onto = /home/mg/.config/opencode/skills/onto -> /home/mg/homonto/content/skills/onto
  + skill.onto-open = /home/mg/.config/opencode/skills/onto-open -> /home/mg/homonto/content/skills/onto-open
  + skill.onto-design = /home/mg/.config/opencode/skills/onto-design -> /home/mg/homonto/content/skills/onto-design
```
### `./homonto apply --yes`
```
claude:
  + skill.onto = /home/mg/.claude/skills/onto -> /home/mg/homonto/content/skills/onto
  + skill.onto-open = /home/mg/.claude/skills/onto-open -> /home/mg/homonto/content/skills/onto-open
  + skill.onto-design = /home/mg/.claude/skills/onto-design -> /home/mg/homonto/content/skills/onto-design
  + skill.onto-build = /home/mg/.claude/skills/onto-build -> /home/mg/homonto/content/skills/onto-build
  + skill.onto-verify = /home/mg/.claude/skills/onto-verify -> /home/mg/homonto/content/skills/onto-verify
  + skill.onto-close = /home/mg/.claude/skills/onto-close -> /home/mg/homonto/content/skills/onto-close
  + skill.onto-fix = /home/mg/.claude/skills/onto-fix -> /home/mg/homonto/content/skills/onto-fix
  + skill.onto-tweak = /home/mg/.claude/skills/onto-tweak -> /home/mg/homonto/content/skills/onto-tweak
opencode:
  + skill.onto = /home/mg/.config/opencode/skills/onto -> /home/mg/homonto/content/skills/onto
  + skill.onto-open = /home/mg/.config/opencode/skills/onto-open -> /home/mg/homonto/content/skills/onto-open
  + skill.onto-design = /home/mg/.config/opencode/skills/onto-design -> /home/mg/homonto/content/skills/onto-design
  + skill.onto-build = /home/mg/.config/opencode/skills/onto-build -> /home/mg/homonto/content/skills/onto-build
  + skill.onto-verify = /home/mg/.config/opencode/skills/onto-verify -> /home/mg/homonto/content/skills/onto-verify
  + skill.onto-close = /home/mg/.config/opencode/skills/onto-close -> /home/mg/homonto/content/skills/onto-close
  + skill.onto-fix = /home/mg/.config/opencode/skills/onto-fix -> /home/mg/homonto/content/skills/onto-fix
  + skill.onto-tweak = /home/mg/.config/opencode/skills/onto-tweak -> /home/mg/homonto/content/skills/onto-tweak
Applied.
```
### `ls -l ~/.claude/skills/ | grep onto`
```
lrwxrwxrwx 1 mg mg   36 Jul  4 13:16 onto -> /home/mg/homonto/content/skills/onto
lrwxrwxrwx 1 mg mg   42 Jul  4 13:16 onto-build -> /home/mg/homonto/content/skills/onto-build
lrwxrwxrwx 1 mg mg   42 Jul  4 13:16 onto-close -> /home/mg/homonto/content/skills/onto-close
lrwxrwxrwx 1 mg mg   43 Jul  4 13:16 onto-design -> /home/mg/homonto/content/skills/onto-design
lrwxrwxrwx 1 mg mg   40 Jul  4 13:16 onto-fix -> /home/mg/homonto/content/skills/onto-fix
lrwxrwxrwx 1 mg mg   41 Jul  4 13:16 onto-open -> /home/mg/homonto/content/skills/onto-open
lrwxrwxrwx 1 mg mg   42 Jul  4 13:16 onto-tweak -> /home/mg/homonto/content/skills/onto-tweak
lrwxrwxrwx 1 mg mg   43 Jul  4 13:16 onto-verify -> /home/mg/homonto/content/skills/onto-verify
```
### `./homonto status`
```
No drift.
```
### `./homonto doctor`
```
warn: `pass` not found on PATH (pass: references will fail)
ok: .claude (Claude Code) config location present
ok: .config/opencode (OpenCode) config location present
ok: skill "onto" present
ok: skill "onto-open" present
ok: skill "onto-design" present
ok: skill "onto-build" present
ok: skill "onto-verify" present
ok: skill "onto-close" present
ok: skill "onto-fix" present
ok: skill "onto-tweak" present
```
### `./homonto plan` (after apply — idempotent)
```
No changes. Everything up to date.
```
### `go test ./...`
```
12 packages ok, 0 failures
```

## Final checks (task 5.3, 2026-07-04)

### Self-containment
```
$ grep -rn "openspec\|comet\|docs/superpowers" content/skills/
(no matches, exit 1)
```

### Symlink load
```
$ test -f ~/.claude/skills/onto/SKILL.md && echo RESOLVES
RESOLVES
```
All eight onto skills were also registered live by Claude Code in the
authoring session (available-skills list), confirming they load via symlink.

### Status / doctor
```
$ ./homonto status
No drift.
$ ./homonto doctor
warn: `pass` not found on PATH (pass: references will fail)
ok: .claude (Claude Code) config location present
ok: .config/opencode (OpenCode) config location present
ok: skill "onto" present   (…and the other seven, all ok)
```

### Regression
```
$ go test ./...
48 passed in 14 packages, 0 failures
```

### Migration audit (build scope)
```
$ grep -rn "openspec/specs\|docs/superpowers" README.md docs/guides docs/specs docs/adr content/
(no matches — live docs reference only new paths)
```
git history preserved across moves (git log --follow works on docs/specs/*.md and docs/roadmap.md).

## Dry-run: full lifecycle (task 5.1, agent-simulated, 2026-07-04)

Fresh-context agent walked open → design → build → verify → close on a
scratch change following the eight skills literally. All 9 checklist items
PASS (preflight-first, zero-active routing, both open gates fresh,
workspace matches contract, approach gate blocks design.md, plan-ready gate
+ commit-per-task + root-cause-first, evidence-based verification, close
merge/numbering/guides/archive semantics, derivation consistent at every
boundary). Scratch dir removed, no tracked file touched.

## Dry-run: presets + drift (task 5.2, agent-simulated, 2026-07-04)

All 8 checklist items PASS (fix skips design, failing-test-first regardless
of tdd, 3+-files upgrade gate; tweak plan-less build, brief-but-mandatory
verification.md, config-key upgrade gate; drift demotion verify→build with
announced correction; deleted state.yaml rebuilt, never fails).

## Defects found by dry-runs — all fixed in the same build phase

1. Derivation table direction reversed ("bottom wins" → "top wins",
   strongest evidence first) in dispatcher + contract (kept identical).
2. Silent gate-skipping on lagging phase: new rule "files win downward,
   gates win upward" — a lagging claim resumes at the unanswered gate.
3. "plan/tasks in progress" row replaced with file-observable
   `Status: Confirmed` marker; verify row keys on verification.md's
   `Result:` line (breaks state.yaml circularity).
4. `guides: waived: <reason>` invalid YAML → quoted scalar everywhere.
5. verify.result enum: deviations recorded in the report, enum stays pass.
6. `decisions.directive` field added for verbatim pre-authorizations.
7. base_ref rebuild = parent of oldest workspace commit; full per-field
   rebuild rules enumerated in the contract.
8. Regression rule for repos with no build/test suite (record the fact).
9. graphify preflight: index is the user's decision; documented fallback.
10. Presets: `Preset: fix|tweak` proposal marker (rebuild keys on it),
    decisions defaulted at open-lite, upgrade thresholds exclude test files.
11. Archived state.yaml keeps `phase: close`; "done" is derived-only.
