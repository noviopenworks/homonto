## Why
ROADMAP E2 / F50 (next safe additive slice): after `status --output json`, extend
the machine-readable pattern to `doctor` (findings). `plan --output json` needs a
structured change-set schema (a design decision on the public shape of a plan) and
is deferred; `doctor`'s output is already a flat findings list, so it is a clean
additive slice.
## What Changes
- `homonto doctor --output text|json` (default text). `json` emits
  `{"findings": [...]}`. No exit-code change.
## Impact
- **Code:** `internal/cli/status.go` (doctorCmd) + test.
- **Spec:** `cli-commands` delta (doctor --output json).
- **Out of scope:** `plan --output json` (change-set schema, design-first); exit-code taxonomy.
