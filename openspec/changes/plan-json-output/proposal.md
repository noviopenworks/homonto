## Why
ROADMAP E2 / F50 (safe additive part): `status` and `doctor` gained `--output json`;
`plan` did not, because a plan's public JSON shape is a contract. Ship a
conservative, safe shape — per-tool visible changes as `{action, key}` (NOT the
Old/New values, which can carry unresolved `${...}` secret tokens), plus remote
repins (digests, non-secret) and warnings. No exit-code change.
## What Changes
- `homonto plan --output text|json` (default text). `json` emits
  `{"changes":[{"tool","changes":[{"action","key"}]}],"repins":[{"name","old","new"}],"warnings":[...]}`.
## Impact
- **Code:** `internal/cli/plan.go` + test.
- **Spec:** `cli-commands` delta (plan --output json).
- **Out of scope:** the versioned exit-code taxonomy (design-first).
