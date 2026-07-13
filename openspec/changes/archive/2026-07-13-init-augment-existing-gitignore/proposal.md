## Why
ROADMAP E2 / finding F52 (safe core): `scaffold.Init` skips an existing `.gitignore`
entirely (`scaffold.go:72`), so a repo that already has one never gets `/.homonto/`
or `.env` ignored — a user can accidentally commit control-plane state or secrets.
## What Changes
- `Init` augments an existing `.gitignore` with any missing homonto entries
  (preserving existing content) and reports created vs updated files.
## Impact
- **Code:** `internal/scaffold/scaffold.go`, `internal/cli/init.go` + test.
- **Spec:** `cli-commands` delta (init augments existing .gitignore).
