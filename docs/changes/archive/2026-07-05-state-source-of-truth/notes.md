# Notes: state-source-of-truth

Incremental checkpoint (compaction recovery). Unconfirmed items are
marked *pending*.

## Confirmed

- 2026-07-05 — Scope bundles NEXT_AGENT gaps #1 (state adoption) and #2
  (status ≠ disk-vs-state). User chose "Bundle #1 + #2" — they share one
  root cause: the non-secret code path ignores `state.json`'s
  `Entry.Applied` hash.
- 2026-07-05 — Full workflow (touches adapter core semantics, changes
  existing spec requirements, needs design decisions). Not a preset.
- 2026-07-05 — **Status semantics** (user gate answer: "Report as pending,
  not drift"): `status` compares disk against `Entry.Applied`. When
  desired≠applied but disk==applied, report "No disk drift" plus a separate
  "N config change(s) awaiting apply" line — pending, not drift.
- 2026-07-05 — **Adoption UX** (user gate answer: "Adopt silently on
  apply"): a declared key already matching disk but absent from state gets
  an `Applied`-hash state entry during apply, with no visible plan/diff
  line.
- 2026-07-05 — Clarification-complete gate answered: "Approve as-is". Name
  confirmed `state-source-of-truth`.
- 2026-07-05 — Split preflight: considered, keep as one change — #1 and #2
  are coupled (one root cause, one state field); splitting leaves semantics
  half-correct.

## Pending

- (verify) verification level from change scale — at onto-verify entry.

## Build

- 2026-07-05 — plan-ready gate answered: **isolation: worktree, execution:
  subagent, tdd: tdd**. No blanket directive — each gate answered fresh.
- Worktree: `/home/mg/homonto-wt/state-source-of-truth`, branch
  `feature/20260705/state-source-of-truth`, base commit 60c095b (workspace),
  parent a72e535 (= base_ref). Workspace removed from main tree; worktree is
  the single source of truth for this change until merge.
- Mid-build refinement: added **Task 3b** (conditional tool-file writes) —
  adopt/noop-only apply must not rewrite the file (opencode comment-strip);
  design.md + tool-adapters delta updated.
- Drift reviewer (after risk:high T6/T7): **no CRITICAL**; core gap closed
  (drift from ObserveHashes-vs-Applied only, config edit → pending not drift),
  secret-safe, deterministic, parity clean. One **MAJOR**: phantom
  non-clearable drift when a recorded key's disk is reconciled out of band to
  the desired value (noop never refreshes stale `Applied`). Fix = **Task 3c**
  (broaden noop→adopt: true noop only when `inState && Applied==hash(disk)`,
  mirroring the secret branch). Design/ADR/apply-pipeline delta updated + new
  stale-Applied scenario. Two minor test-coverage gaps folded into Task 3c/9.
- Adapter reviewer (after risk:high T2/T3/T3b), commits 60c095b..0281087:
  **no CRITICAL**; adopt logic secret-safe, hash-correct, parity clean,
  conditional-write flags complete. Findings map to upcoming tasks — Major#1
  (CLI empty output for adopt-only) → Task 5 (also fix `HasChanges` to
  visible-only so `plan` stays silent); Minor#2 (engine resolves adopt) →
  Task 4. Both accepted as pending-task work, not defects.

## Design ground-truth (verified 2026-07-05, direct reads)

- `apply.go:42-45` **short-circuits** when `plan.HasChanges` is false
  (any action ≠ noop). ⇒ if adoption were a plain `noop`, apply would
  short-circuit and never adopt when nothing else changed — which is the
  primary adoption scenario. Adoption must count as apply-time *work*.
- `Entry.Applied = secret.Hash(canonical(resolved-on-disk))` uniformly for
  secret & non-secret (`claude.go:225`, `opencode.go:203`) ⇒ a disk-vs-
  Applied drift check works uniformly across both.
- adopt needs **no disk read at apply**: it fires only when non-secret
  `canonical(disk)==canonical(want)`, so `Applied=hash(canonical(resolve(want)))`
  — same value a normal write stores, minus the file write.
- opencode plugins are **array membership** (`opencode.go:73-79`,
  `arrayHas`), not keyed values ⇒ plugin drift is present/absent only.
  claude plugins are keyed (`enabledPlugins` object, value `true`).
- Owned skills are re-recorded on every apply (`claude.go:238-244`,
  `opencode.go:213-219`) ⇒ never unadopted; adoption gap is only
  mcp/setting/plugin keys.

## Grounding

- graphify-out/ present at repo root; no `.codegraph`. graphify skill
  loadable. Codebase claims below were verified against actual source via
  direct file reads (Explore agents), which satisfies the grounding
  requirement.
- Gap #1 confirmed: non-secret matching key → `Action: "noop"` with no
  `st.Set`; noop branch ignores `inState`.
  `internal/adapter/claude/claude.go:82-108` (noop decision :88-90);
  `internal/adapter/opencode/opencode.go:130-153` (`planKey`, noop :136-137).
  Apply skips noop before state write: `claude.go:179-182`,
  `opencode.go:160-163`, `internal/engine/engine.go:83-91`.
- Pruning iterates only `st.Keys(tool)`
  (`claude.go:120-135`, `opencode.go:91-112`) → unrecorded keys are
  unpruneable.
- Gap #2 confirmed: `engine.Drift` reuses `Plan()`
  (`internal/engine/status.go:10-36`); non-secret Plan compares disk vs
  *desired*, never against `Entry.Applied`. Secret branch already compares
  `Entry.Applied == secret.Hash(canonical(disk))` (`claude.go:101-106`).
- State model: `internal/state/state.go` — `Entry{Desired, Applied}`,
  `Applied` = sha256 of resolved on-disk value; `Save` writes
  `.homonto/state.json` atomically. `Set` :62, `Get` :70, `Keys` :76,
  `Delete` :86.
- `status` CLI: `internal/cli/status.go:10-38` calls `e.Drift()`.
- Existing drift tests (`internal/engine/status_test.go`) only mutate disk
  out of band; none cover the gap (config edit, disk unchanged, must NOT be
  drift).

## Build outcome (2026-07-05)

- All tasks done (1, 2, 3, 3b, 3c, 4, 5, 6, 7[+8 merged], 9). 12 code/doc
  commits on `feature/20260705/state-source-of-truth`.
- Validation: `go build`, `go vet`, `go test ./...` (125), `go test -race`
  (125), `gofmt -l internal/` (empty) — all green.
- Manual smoke (real binary, temp HOME): adoption "Reconciled 1..." +
  settings.json byte-identical + state records key; config edit → "1 config
  change(s) awaiting apply" (pending, not drift); disk edit → drifted;
  adopt-only plan → "No changes".
- **Final holistic reviewer**: change is correct and spec-complete — all 10
  delta-spec scenarios backed by code + tests; no CRITICAL/major. Accepted
  minors (non-blocking, for verify to note/accept):
  1. status.go:50/56 wording: a key that is BOTH de-declared AND drifted/missing
     says "will reset"/"deleted out of band" though apply will actually prune it
     — cosmetic edge combo only.
  2. A broken adapter emits two warnings (Plan "skipped" + ObserveHashes "drift
     skipped") — redundant, harmless; drift correctly excludes it.
  3. Coverage nits: CLI-level "second apply No changes" after adoption, env-
     bearing MCP steady-state noop, and brand-new create counted as pending are
     each proven at another layer but not their own dedicated test.

## Approaches  <!-- design phase -->

Two coupled sub-decisions (adoption mechanism, drift mechanism) bundled into
coherent overall approaches:

- **Approach 1 — First-class `adopt` action + drift decoupled via adapter
  observation (RECOMMENDED).**
  - Adoption: new invisible `adopt` action emitted by Plan for a declared,
    non-secret, disk==desired key *not yet in state*. Renders no diff line.
    apply.go runs even when adoption is the only work (state-only, no
    confirm prompt, one-line "Reconciled N …" summary); plan stays silent.
    Adapter.Apply records state (`st.Set`) without writing files.
  - Drift: new adapter capability exposing per-key disk **hashes**
    (secret-safe — only hashes leave the adapter). `engine.Status()`
    computes disk-vs-`Applied` drift and a `pending` count (Plan visible
    changes whose key is not itself drifted). status prints drift lines +
    "N config change(s) awaiting apply".
  - Pros: honest layering (Plan=intent, adopt=explicit, drift=own
    computation); fixes the architectural root cause (drift no longer
    piggybacks Plan); secret-safe. Cons: touches the ~4 action-literal
    sites + one new interface method + apply.go flow split.

- **Approach 2 — Minimal surface: overload `noop`, thread disk hash through
  `Change`, drift stays Plan-derived.**
  - Adoption: no new action; apply.go always calls Apply (reconcile);
    adapter adopts noop-keys-not-in-state. Drift: add `Change.DiskHash` to
    every Change; engine.Drift compares to `Applied`; pending from actions.
  - Pros: no interface method, fewer concepts. Cons: overloads noop (less
    honest); "No changes" path now writes state (muddy); drift still
    structurally coupled to Plan (de-declared/orphan keys have no clean
    disk hash); `Change` bloat.

- **Approach 3 — Adapter-owned StatusReport.** Each adapter returns a full
  {drift, pending, adoptions} report; engine aggregates. Pros: maximal
  encapsulation. Cons: largest interface change; duplicates pending logic
  per adapter; over-engineered for two tools.

RECOMMENDED: Approach 1. **CONFIRMED 2026-07-05** — user picked "Approach 1
(recommended)". design.md written Status: Confirmed; ADR drafts
`adopt-preexisting-resources-into-state`, `drift-from-disk-vs-state`; delta
specs for apply-pipeline (ADDED adoption, MODIFIED Drift detection) and
tool-adapters (ADDED adapter adoption).
