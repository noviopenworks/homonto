## Why
ROADMAP E2 / finding F46 (safe core): a catalog version bump materializes silently
during `apply`, and `doctor` does not report a pending catalog upgrade — the
recorded catalog version (`state.json`) can lag the embedded catalog version with
no visible signal. doctor should surface the mismatch.
## What Changes
- `homonto doctor` reports a finding when the recorded catalog version differs
  from the embedded catalog version (a catalog upgrade is pending; run `apply`).
## Impact
- **Code:** `internal/engine/status.go` (Doctor) + a small helper + test.
- **Spec:** `cli-commands` delta (doctor reports catalog-version mismatch).
- **Out of scope:** surfacing the catalog upgrade in `plan` output (F46 remainder).
