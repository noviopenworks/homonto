## Why

ROADMAP N2 (the last "Now" item). N1 made the onto binary own real evidence
fields (verify.result, guides, close.merged, isolation) but the gates still check
only artifact existence + task checkboxes: `onto close` gates phase/deps/clean-
worktree, NOT `verify.result == pass`, resolved `guides`, or `close.merged`
(`internal/ontocli/close.go`), so empty/unverified work can still archive. B1
means the binary enforces that the phase's evidence *tokens* are present and
well-formed. This change makes the binary's advance/close gates semantic over the
tokens N1 added — workflow-aware, so fix/tweak presets are gated on a reduced set.

## What Changes

- **`onto close` requires real close evidence:** for a `full` workflow, close
  SHALL additionally require `verify.result == pass`, `close.merged == true`, and
  `guides` resolved (`updated` or `waived:<reason>`, not `pending`/empty), on top
  of the existing phase/deps/clean-worktree gates. `fix`/`tweak` presets are gated
  on the reduced set they actually produce (`verify.result == pass` +
  `close.merged`; guides not required).
- **`onto advance` gates on phase evidence:** leaving `verify` SHALL require
  `verify.result == pass` (not `pending`/`fail`); entering `build` SHALL require
  `isolation` chosen (branch/worktree) so planning never lands unisolated
  (workflow-safety, F15 at the binary boundary).
- Clear, actionable errors naming the missing evidence.

## Impact

- **Code:** `internal/ontocli/close.go`, `internal/ontocli/advance.go`,
  possibly a small `ontostate` helper for "guides resolved" / workflow-aware
  required-evidence + tests.
- **Spec:** `onto-binary` delta (advance/close evidence gates, workflow-aware).
- **Out of scope (separable N2 follow-ups):** comet verification-scale-by-risk +
  non-waivable finding classes + bundling skeptic/reviewer subagents (F11/F12);
  skill-plane recovery/atomic-task-bookkeeping (F16/F17); a full dep resolver
  with cycle detection (F10) beyond the existing DepsResolved. These are noted,
  not implemented here.
