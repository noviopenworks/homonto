# Proposal: state-source-of-truth

## Why

`state.json` already stores, per managed key, an `Entry.Applied` field: the
sha256 of the last-applied resolved value written to disk. This is exactly
the data needed to answer "does disk still match what we last applied?" —
but the **non-secret** code path never consults it. Only the secret-bearing
branch compares against `Entry.Applied`; non-secret keys are decided purely
by comparing the current on-disk value against the current *desired* value.

That single leak produces the two highest-priority gaps in the deep-review
handoff (`docs/NEXT_AGENT.md` #1 and #2):

1. **Adoption gap.** A declared key whose on-disk value already equals
   desired yields `Action: "noop"` and never calls `st.Set`, so it never
   enters `state.json`. Imported or manually pre-existing resources look
   managed but stay invisible to pruning (the prune loop iterates only
   `st.Keys(tool)`) and to state-gated drift checks.
2. **Drift gap.** `engine.Drift` reuses the desired-driven `Plan()`, so
   editing `homonto.toml` makes a key appear "drifted" even when disk is
   unchanged since the last apply. `status` conflates un-applied config
   edits with real out-of-band disk drift.

Both are the same missing behavior — trusting `Entry.Applied` as the
authority — so they are fixed together.

## What Changes

- Adapters record state for a **declared key that already matches disk but
  is absent from state**: on apply it gets an `Entry` (`Desired` = desired,
  `Applied` = hash of the on-disk resolved value) — silently, with no plan
  diff line. Adopted keys become pruneable and drift-visible. Applies to
  both the claude and opencode adapters identically.
- **`status` / drift is redesigned to compare disk against `Entry.Applied`**
  for non-secret keys (matching what the secret branch already does), not
  against desired.
- **`status` distinguishes pending from drift**: when desired≠applied but
  disk still matches applied, it reports "No disk drift" plus a separate
  "N config change(s) awaiting apply" line, rather than reporting the
  config edit as drift.
- Adoption only ever covers keys the user **declares** in config — never
  arbitrary undeclared on-disk values.

No breaking change to the `state.json` on-disk format (`Applied` already
exists); no CLI flag changes.

## Capability Impact

- **Modified**: `apply-pipeline` — the idempotency/status/drift requirements
  change: adopt matching declared keys into state on apply; drift is
  disk-vs-`Applied`; status separates pending config changes from disk
  drift. (delta required — `docs/specs/apply-pipeline.md`)
- **Modified**: `tool-adapters` — the plan/apply requirements change: a
  declared key matching disk but absent from state is adopted (state
  recorded) rather than left untracked; adapter parity for claude and
  opencode. (delta required — `docs/specs/tool-adapters.md`)
- Untouched: `config-model`, `secret-references` (secret semantics already
  correct and preserved), `cli-commands` (command surface unchanged),
  `onto-workflow`.

## Not split

#1 and #2 are two halves of one behavior: making `Entry.Applied` the
authority for both the managed set (adoption/pruning) and drift detection.
They touch the same state field and the same non-secret adapter branch;
fixing one alone leaves the semantics half-correct (adopted keys still
drift-invisible, or drift correct but imported keys still unpruneable).
Kept as one change.

## Grounding

graphify-out/ present; no `.codegraph`. Claims verified against actual
source via direct file reads (exploration). Key anchors: noop-without-state
at `internal/adapter/claude/claude.go:88-90` and
`internal/adapter/opencode/opencode.go:136-137`; drift reuse at
`internal/engine/status.go:10-36`; `Entry.Applied` at
`internal/state/state.go:17-25`; prune loop over `st.Keys` at
`claude.go:120-135` / `opencode.go:91-112`. Full anchor list in notes.md
Grounding.

## Impact

- Source: `internal/adapter/claude/claude.go`,
  `internal/adapter/opencode/opencode.go`, `internal/engine/status.go`,
  `internal/cli/status.go` (pending-apply output).
- Tests: adapter plan/apply/pruning tests, engine drift tests
  (`internal/engine/status_test.go` — add the currently-missing
  "config edit is not disk drift" and "adoption" cases).
- Specs: `docs/specs/apply-pipeline.md`, `docs/specs/tool-adapters.md`.
- Risk: apply currently skips `noop` before `st.Set`; the adopt path must
  record state without introducing a phantom write or a visible diff line —
  a design decision. Must not accidentally route secret keys through the
  non-secret adopt path.
- Out of scope (separate changes): NEXT_AGENT #3–#8.
