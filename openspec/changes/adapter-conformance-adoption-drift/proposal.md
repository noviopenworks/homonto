## Why
ROADMAP E3 / F55 (slice 2): extend the shared adapter conformance suite beyond the
core contract to the next properties every adapter must honor — drift detection/
reset and malformed-doc safety — so the suite catches more divergence uniformly.
## What Changes
- The conformance suite gains, for claude + opencode: (a) drift — a managed file
  changed out-of-band is reported by ObserveHashes as differing from Entry.Applied,
  and a re-Apply resets it; (b) malformed-doc safety — a pre-existing malformed
  tool doc does not crash Plan/Apply (error or recover, never panic).
## Impact
- **Code:** `internal/adapter/conformance/conformance_test.go` (extend, test-only).
- **Spec:** `tool-adapters` delta (conformance covers drift + malformed-doc safety).
- **Out of scope:** adoption of foreign content, secret redaction, conflict safety, codex, consolidation.
