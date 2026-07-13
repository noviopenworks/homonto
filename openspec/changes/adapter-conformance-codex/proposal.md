## Why
ROADMAP E3 / F55 (completes adapter coverage): the conformance suite covers claude
and opencode but not codex, whose surface is reduced (MCP-only per the pilot). Add
codex to the shared suite, applying the checks its surface supports and explicitly
skipping (with a documented reason) those it does not, so every shipped adapter is
covered uniformly and codex's reduced-surface expectations are pinned.
## What Changes
- The conformance table gains a codex row; each conformance check either runs for
  codex or is explicitly skipped with a comment stating why (reduced MCP-only surface).
## Impact
- **Code:** `internal/adapter/conformance/conformance_test.go` (add codex, test-only).
- **Spec:** `codex-adapter` delta (codex is covered by the shared conformance suite for its supported surface).
- **Out of scope:** broadening codex beyond MCP; the structproj consolidation.
