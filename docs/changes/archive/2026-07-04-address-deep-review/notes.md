# Notes: address-deep-review

Incremental checkpoint (compaction recovery). Unconfirmed items are
marked *pending*.

## Confirmed

- 2026-07-04 requirements: the deep review (docs/reviews/
  2026-07-04-deep-review.md) is the requirements document — user
  dictated it be written down, then set goal "implentation of things is
  finished". Scope = review priority items 1–7 except pushing to origin
  (user twice chose not to push; remains excluded until asked).
- 2026-07-04 clarification/artifact-review/approach/plan-ready gates:
  pre-answered by the goal directive + the review's prescriptive fixes
  (recorded verbatim in state.yaml decisions.directive).
- Approach (from the review, confirmed by endorsement): fix in place —
  correct Claude MCP schema to {type,command:string,args}, import reads
  string+args, redact-when-unknown for missing state, mode-preserving
  0600-default atomic writes with fsync, real pruning via delete action
  + state-driven orphan detection (incl. skill links), sjson path
  escaping + skill-name validation, sorted plan output, per-adapter
  state save, memoized resolver, MIT LICENSE, GitHub Actions CI,
  var Version, README honesty edits, conformance fixtures from real
  tool files; onto v2.1: tweak covers small features, preflight
  warn-not-halt, ADR 0007 errata, close rewrites ADR links before
  archive, guide sync.
- execution: subagent (coordinator protocol), tdd: tdd (bug fixes =
  failing test first), isolation: branch
  (feature/20260704/address-deep-review).

## Pending

- Push to origin + repo visibility + tag: excluded from this change;
  ask user after close.
- LICENSE chosen as MIT (personal-OSS default) — flag at close summary
  for user override.

## Grounding

- The review itself: four fresh-context reviewers with file:line
  evidence, two findings reproduced against the built binary, MCP
  schema verified against the live ~/.claude.json
  ({"type":"stdio","command":"codegraph","args":[...]}).

## Approaches

- Review-prescribed fixes — **CONFIRMED 2026-07-04 via directive** (the
  review contains the alternatives analysis; re-deriving them would be
  ceremony).

## Verify round 1 (2026-07-04) — FAIL → fixed per directive triage

Conformance: core fixes held empirically (schema, pruning, escaping,
redaction, modes, memoization, per-adapter state); REFUTED: import parse
silence, import Claude/OpenCode over-claim, adapter-naming-by-accident.
Robustness: 10 findings — HIGH: numeric/empty names corrupt to arrays;
mediums: url-server import loss, symlink-replaced-by-file writes, tweak
off-by-one + fix/tweak ceremony inversion, stale "preflight guarantees"
lines; lows: deleted-link drift, plugin scalar guard, relink dead-end.
All fixed (Go commit e0ea17c + markdown round); accepted deviations:
two-phase token equivalence for quote/backslash env names (exotic),
kill-9 temp-file stranding (pre-existing).
