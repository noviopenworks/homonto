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
