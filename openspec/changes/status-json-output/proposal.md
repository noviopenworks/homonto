## Why
ROADMAP E2 / finding F50 (safe additive first slice): homonto has no
machine-readable output, so automation must scrape human text. Start the stable
automation contract with a read-only, additive `--output json` on `status` (drift
+ pending + warnings), establishing the pattern. The full F50 — a versioned
exit-code taxonomy and JSON across every command — is a public contract that needs
a design phase and is out of scope here.
## What Changes
- `homonto status --output text|json` (default text). `json` emits
  `{"drift": [...], "pending": N, "warnings": [...]}`. No exit-code change.
## Impact
- **Code:** `internal/cli/status.go` + test.
- **Spec:** `cli-commands` delta (status --output json).
- **Out of scope:** exit-code taxonomy; json for plan/apply/doctor (F50 remainder, design-first).
